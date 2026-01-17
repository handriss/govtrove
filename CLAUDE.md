# Claude Code Guidelines for OpScout

## Deployment Policy

**Never deploy automatically.** All deployments must be done by the user after review.

This includes:
- `terraform apply` - prepare the plan, but do not apply
- `docker push` to ECR - build images, but do not push
- Any ECS task triggers - do not trigger tasks
- Any infrastructure changes that affect production

Always stop and present the deployment command for user review and manual execution.

## Code Style

### Comments

Comments are for intent, not narration.
Write comments only when explaining why something exists (constraints, tradeoffs, landmines, or intentional ugliness).
If a comment explains what the code does, rewrite the code instead.
Assume comments rot; structure and names must carry meaning.
