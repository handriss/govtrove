package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"

	middleware "github.com/handriss/govtrove/api/internal/middleware"
	"github.com/handriss/govtrove/api/internal/repository"
)

type StripeHandler struct {
	userRepo      *repository.UserRepository
	promoRepo     *repository.PromoCodeRepository
	sc            *stripe.Client
	webhookSecret string
	priceMonthly  string
	appURL        string
	logger        *slog.Logger
}

func NewStripeHandler(
	userRepo *repository.UserRepository,
	promoRepo *repository.PromoCodeRepository,
	sc *stripe.Client,
	webhookSecret string,
	priceMonthly string,
	appURL string,
	logger *slog.Logger,
) *StripeHandler {
	return &StripeHandler{
		userRepo:      userRepo,
		promoRepo:     promoRepo,
		sc:            sc,
		webhookSecret: webhookSecret,
		priceMonthly:  priceMonthly,
		appURL:        appURL,
		logger:        logger,
	}
}

func (h *StripeHandler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	workosID := middleware.UserIDFromContext(r.Context())
	if workosID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.userRepo.GetByWorkOSID(r.Context(), workosID)
	if err != nil || user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	if user.Plan == "pro" && !user.FreeForever {
		http.Error(w, "already subscribed", http.StatusConflict)
		return
	}

	// Parse optional promo code from request body
	var body struct {
		PromoCode string `json:"promo_code"`
	}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&body)
	}

	customerID := ""
	if user.StripeCustomerID != nil {
		customerID = *user.StripeCustomerID
	} else {
		cust, custErr := h.sc.V1Customers.Create(r.Context(), &stripe.CustomerCreateParams{
			Email: stripe.String(user.Email),
			Name:  stripe.String(user.FirstName + " " + user.LastName),
			Metadata: map[string]string{
				"govtrove_user_id": fmt.Sprintf("%d", user.ID),
				"workos_id":        user.WorkOSID,
			},
		})
		if custErr != nil {
			h.logger.Error("stripe customer create failed", "error", custErr)
			http.Error(w, "failed to create customer", http.StatusInternalServerError)
			return
		}
		customerID = cust.ID
		if err := h.userRepo.SetStripeCustomerID(r.Context(), user.ID, customerID); err != nil {
			h.logger.Error("failed to save stripe customer id", "error", err)
		}
	}

	params := &stripe.CheckoutSessionCreateParams{
		Customer:                  stripe.String(customerID),
		Mode:                     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		BillingAddressCollection: stripe.String(string(stripe.CheckoutSessionBillingAddressCollectionRequired)),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{
				Price:    stripe.String(h.priceMonthly),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(h.appURL + "/profile?upgraded=1"),
		CancelURL:  stripe.String(h.appURL + "/profile"),
	}

	if body.PromoCode != "" && h.promoRepo != nil {
		promo, promoErr := h.promoRepo.GetByCode(r.Context(), body.PromoCode)
		if promoErr != nil {
			h.logger.Error("promo code lookup failed", "error", promoErr)
		} else if promo == nil {
			http.Error(w, "invalid promo code", http.StatusBadRequest)
			return
		} else if promo.RedeemedBy != nil {
			http.Error(w, "promo code already used", http.StatusBadRequest)
			return
		} else if promo.ExpiresAt != nil && promo.ExpiresAt.Before(time.Now()) {
			http.Error(w, "promo code expired", http.StatusBadRequest)
			return
		} else {
			params.Discounts = []*stripe.CheckoutSessionCreateDiscountParams{
				{PromotionCode: stripe.String(promo.StripePromoID)},
			}
		}
	}

	sess, err := h.sc.V1CheckoutSessions.Create(r.Context(), params)
	if err != nil {
		h.logger.Error("checkout session create failed", "error", err)
		http.Error(w, "failed to create checkout session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": sess.URL})
}

func (h *StripeHandler) CreatePortalSession(w http.ResponseWriter, r *http.Request) {
	workosID := middleware.UserIDFromContext(r.Context())
	if workosID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.userRepo.GetByWorkOSID(r.Context(), workosID)
	if err != nil || user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	if user.StripeCustomerID == nil {
		http.Error(w, "no billing account", http.StatusBadRequest)
		return
	}

	sess, err := h.sc.V1BillingPortalSessions.Create(r.Context(), &stripe.BillingPortalSessionCreateParams{
		Customer:  user.StripeCustomerID,
		ReturnURL: stripe.String(h.appURL + "/profile"),
	})
	if err != nil {
		h.logger.Error("portal session create failed", "error", err)
		http.Error(w, "failed to create portal session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": sess.URL})
}

func (h *StripeHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	event, err := webhook.ConstructEventWithOptions(body, r.Header.Get("Stripe-Signature"), h.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		h.logger.Warn("stripe webhook signature verification failed", "error", err)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	already, dbErr := h.userRepo.CheckAndRecordWebhookEvent(ctx, event.ID, string(event.Type))
	if dbErr != nil {
		h.logger.Warn("webhook idempotency check failed, processing anyway", "error", dbErr)
	} else if already {
		h.logger.Debug("duplicate webhook skipped", "event_id", event.ID)
		w.WriteHeader(http.StatusOK)
		return
	}

	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted:
		h.handleCheckoutCompleted(ctx, &event)
	case stripe.EventTypeCustomerSubscriptionUpdated:
		h.handleSubscriptionUpdated(ctx, &event)
	case stripe.EventTypeCustomerSubscriptionDeleted:
		h.handleSubscriptionDeleted(ctx, &event)
	case stripe.EventTypeInvoicePaymentFailed:
		h.logger.Warn("stripe invoice payment failed", "event_id", event.ID)
	case stripe.EventTypeInvoicePaid:
		h.logger.Debug("stripe invoice paid", "event_id", event.ID)
	default:
		h.logger.Debug("unhandled stripe event", "type", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *StripeHandler) handleCheckoutCompleted(ctx context.Context, event *stripe.Event) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		h.logger.Error("failed to unmarshal checkout session", "error", err)
		return
	}

	if session.Customer == nil {
		h.logger.Warn("checkout session has no customer")
		return
	}

	user, err := h.userRepo.GetByStripeCustomerID(ctx, session.Customer.ID)
	if err != nil || user == nil {
		h.logger.Warn("no user for stripe customer", "customer_id", session.Customer.ID, "error", err)
		return
	}

	var subID *string
	if session.Subscription != nil {
		subID = &session.Subscription.ID
	}

	status := "active"
	if err := h.userRepo.UpdateSubscription(ctx, user.ID, "pro", subID, &status, false, nil); err != nil {
		h.logger.Error("failed to upgrade user plan", "user_id", user.ID, "error", err)
	} else {
		h.logger.Info("user upgraded to pro", "user_id", user.ID, "email", user.Email)
	}

	// Track promo code redemption
	if h.promoRepo != nil && session.ID != "" {
		h.trackPromoRedemption(ctx, session.ID, user.ID)
	}
}

func (h *StripeHandler) trackPromoRedemption(ctx context.Context, sessionID string, userID int) {
	expanded, err := h.sc.V1CheckoutSessions.Retrieve(ctx, sessionID, &stripe.CheckoutSessionRetrieveParams{
		Expand: []*string{stripe.String("discounts.promotion_code")},
	})
	if err != nil {
		h.logger.Debug("could not expand checkout session for promo tracking", "error", err)
		return
	}

	for _, d := range expanded.Discounts {
		if d.PromotionCode == nil {
			continue
		}
		promoID := d.PromotionCode.ID
		row, err := h.promoRepo.GetByStripePromoID(ctx, promoID)
		if err != nil {
			h.logger.Error("promo lookup failed in webhook", "stripe_promo_id", promoID, "error", err)
			continue
		}
		if row == nil {
			continue
		}
		if err := h.promoRepo.MarkRedeemed(ctx, row.ID, userID); err != nil {
			h.logger.Error("failed to mark promo redeemed", "promo_id", row.ID, "error", err)
		} else {
			h.logger.Info("promo code redeemed", "promo_id", row.ID, "code", row.Code, "user_id", userID)
		}
	}
}

func (h *StripeHandler) handleSubscriptionUpdated(ctx context.Context, event *stripe.Event) {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		h.logger.Error("failed to unmarshal subscription", "error", err)
		return
	}

	if sub.Customer == nil {
		return
	}

	user, err := h.userRepo.GetByStripeCustomerID(ctx, sub.Customer.ID)
	if err != nil || user == nil {
		h.logger.Warn("no user for stripe customer", "customer_id", sub.Customer.ID)
		return
	}

	plan := "free"
	switch sub.Status {
	case stripe.SubscriptionStatusActive, stripe.SubscriptionStatusTrialing, stripe.SubscriptionStatusPastDue:
		plan = "pro"
	}

	var periodEnd *time.Time
	if len(sub.Items.Data) > 0 && sub.Items.Data[0].CurrentPeriodEnd > 0 {
		t := time.Unix(sub.Items.Data[0].CurrentPeriodEnd, 0)
		periodEnd = &t
	}

	status := string(sub.Status)
	subID := sub.ID
	if err := h.userRepo.UpdateSubscription(ctx, user.ID, plan, &subID, &status, sub.CancelAtPeriodEnd, periodEnd); err != nil {
		h.logger.Error("failed to update subscription", "user_id", user.ID, "plan", plan, "error", err)
	} else {
		h.logger.Info("subscription updated", "user_id", user.ID, "plan", plan, "status", sub.Status, "cancel_at_period_end", sub.CancelAtPeriodEnd)
	}
}

func (h *StripeHandler) handleSubscriptionDeleted(ctx context.Context, event *stripe.Event) {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		h.logger.Error("failed to unmarshal subscription", "error", err)
		return
	}

	if sub.Customer == nil {
		return
	}

	user, err := h.userRepo.GetByStripeCustomerID(ctx, sub.Customer.ID)
	if err != nil || user == nil {
		h.logger.Warn("no user for stripe customer", "customer_id", sub.Customer.ID)
		return
	}

	status := "canceled"
	if err := h.userRepo.UpdateSubscription(ctx, user.ID, "free", nil, &status, false, nil); err != nil {
		h.logger.Error("failed to downgrade plan", "user_id", user.ID, "error", err)
	} else {
		h.logger.Info("subscription deleted, downgraded to free", "user_id", user.ID)
	}
}
