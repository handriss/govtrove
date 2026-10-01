package bulkcsv

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Validators mirror the latest new_file ETag/Last-Modified held in bulk_csv_log,
// copied to S3 so a no-op poll can ask SAM.gov "changed?" without waking Neon.
type Validators struct {
	ETag         string `json:"etag"`
	LastModified string `json:"last_modified"`
}

type ValidatorObjects interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

func validatorsKey(source string) string {
	return "raw/csv-validators/" + source + ".json"
}

// LoadValidators reports ok=false for any failure; without the S3 permission
// ListBucket a missing key surfaces as AccessDenied, not NotFound.
func LoadValidators(ctx context.Context, c ValidatorObjects, bucket, source string) (Validators, bool) {
	out, err := c.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(validatorsKey(source))})
	if err != nil {
		return Validators{}, false
	}
	defer out.Body.Close()
	var v Validators
	if err := json.NewDecoder(out.Body).Decode(&v); err != nil {
		return Validators{}, false
	}
	return v, v.ETag != "" || v.LastModified != ""
}

func SaveValidators(ctx context.Context, c ValidatorObjects, bucket, source string, v Validators) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = c.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(validatorsKey(source)),
		Body:        bytes.NewReader(body),
		ContentType: aws.String("application/json"),
	})
	return err
}

// NotModified is deliberately conservative: anything other than a clean 304
// sends the caller down the full DB-backed path.
func NotModified(ctx context.Context, src Source, v Validators) bool {
	if src.UseRangeProbe || (v.ETag == "" && v.LastModified == "") {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, rangeRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
	if err != nil {
		return false
	}
	if v.ETag != "" {
		req.Header.Set("If-None-Match", v.ETag)
	}
	if v.LastModified != "" {
		req.Header.Set("If-Modified-Since", v.LastModified)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<10))
	resp.Body.Close()
	return resp.StatusCode == http.StatusNotModified
}
