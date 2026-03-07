import { ExternalLink } from 'lucide-react';
import AdminLayout from '../components/AdminLayout';

const REGION = 'us-east-1';
const ACCOUNT = '341115025444';

interface LinkItem {
  label: string;
  url: string;
  description: string;
}

interface LinkGroup {
  title: string;
  items: LinkItem[];
}

const groups: LinkGroup[] = [
  {
    title: 'Dashboards',
    items: [
      {
        label: 'CloudWatch Dashboard',
        url: `https://${REGION}.console.aws.amazon.com/cloudwatch/home?region=${REGION}#dashboards/dashboard/govtrove-dashboard`,
        description: 'API requests, errors, pipeline executions, Lambda performance, alarms, and billing — all in one view',
      },
      {
        label: 'Sentry (Frontend)',
        url: 'https://govtrove.sentry.io',
        description: 'Frontend error tracking and performance monitoring',
      },
    ],
  },
  {
    title: 'API & MCP',
    items: [
      {
        label: 'App Runner — API',
        url: `https://${REGION}.console.aws.amazon.com/apprunner/home?region=${REGION}#/services/dashboard?service_arn=arn:aws:apprunner:${REGION}:${ACCOUNT}:service/govtrove-api/b2eacce5bed84b30b0a8bd06048b1d5d`,
        description: 'API service metrics, deployments, and configuration',
      },
      {
        label: 'App Runner — MCP',
        url: `https://${REGION}.console.aws.amazon.com/apprunner/home?region=${REGION}#/services/dashboard?service_arn=arn:aws:apprunner:${REGION}:${ACCOUNT}:service/govtrove-mcp/f0b485cdc51d4cea8ad24c3ae08a32eb`,
        description: 'MCP server metrics, deployments, and configuration',
      },
    ],
  },
  {
    title: 'Pipeline',
    items: [
      {
        label: 'Step Functions',
        url: `https://${REGION}.console.aws.amazon.com/states/home?region=${REGION}#/statemachines/view/arn:aws:states:${REGION}:${ACCOUNT}:stateMachine:govtrove-pipeline`,
        description: 'Pipeline state machine — execution history and graph',
      },
      {
        label: 'Pipeline DLQ',
        url: `https://${REGION}.console.aws.amazon.com/sqs/v3/home?region=${REGION}#/queues/https%3A%2F%2Fsqs.${REGION}.amazonaws.com%2F${ACCOUNT}%2Fgovtrove-pipeline-dlq`,
        description: 'Dead letter queue for failed pipeline messages',
      },
    ],
  },
  {
    title: 'Logs',
    items: [
      {
        label: 'Lambda Logs (all)',
        url: `https://${REGION}.console.aws.amazon.com/cloudwatch/home?region=${REGION}#logsV2:log-groups$3FlogGroupNameFilter$3D/aws/lambda/govtrove`,
        description: 'Log groups for all pipeline Lambda functions',
      },
      {
        label: 'App Runner API Logs',
        url: `https://${REGION}.console.aws.amazon.com/cloudwatch/home?region=${REGION}#logsV2:log-groups$3FlogGroupNameFilter$3D/aws/apprunner/govtrove-api`,
        description: 'Application and system logs for the API service',
      },
    ],
  },
  {
    title: 'Alarms & Billing',
    items: [
      {
        label: 'CloudWatch Alarms',
        url: `https://${REGION}.console.aws.amazon.com/cloudwatch/home?region=${REGION}#alarmsV2:?~(search~'govtrove)`,
        description: 'Pipeline failures, 4xx/5xx errors, request spikes, and billing alarms',
      },
      {
        label: 'Cost Explorer',
        url: `https://${REGION}.console.aws.amazon.com/costmanagement/home?region=${REGION}#/cost-explorer`,
        description: 'Detailed cost breakdown by service',
      },
      {
        label: 'Billing Dashboard',
        url: 'https://us-east-1.console.aws.amazon.com/billing/home#/',
        description: 'Current month charges and payment history',
      },
    ],
  },
  {
    title: 'Infrastructure',
    items: [
      {
        label: 'CloudFront — Frontend',
        url: `https://us-east-1.console.aws.amazon.com/cloudfront/v4/home#/distributions/EMMU4U75BLSZ6`,
        description: 'app.govtrove.com distribution — cache stats and error rates',
      },
      {
        label: 'CloudFront — Landing',
        url: `https://us-east-1.console.aws.amazon.com/cloudfront/v4/home#/distributions/E30S7Z01Z2AERU`,
        description: 'govtrove.com distribution',
      },
      {
        label: 'S3 Data Bucket',
        url: `https://s3.console.aws.amazon.com/s3/buckets/govtrove-data?region=${REGION}`,
        description: 'Pipeline CSV downloads and bulk data',
      },
      {
        label: 'ECR Repositories',
        url: `https://${REGION}.console.aws.amazon.com/ecr/repositories?region=${REGION}`,
        description: 'Docker image repositories for API and MCP',
      },
    ],
  },
  {
    title: 'External Services',
    items: [
      {
        label: 'Neon Console',
        url: 'https://console.neon.tech/app/projects/lively-brook-39670028',
        description: 'PostgreSQL database — branches, queries, and connection pooling',
      },
      {
        label: 'Cloudflare',
        url: 'https://dash.cloudflare.com',
        description: 'DNS, WAF rules, bot protection for api.govtrove.com',
      },
      {
        label: 'WorkOS Dashboard',
        url: 'https://dashboard.workos.com',
        description: 'Authentication — users, SSO, and session management',
      },
      {
        label: 'Stripe Dashboard',
        url: 'https://dashboard.stripe.com',
        description: 'Subscriptions, payments, and billing',
      },
      {
        label: 'Resend',
        url: 'https://resend.com/overview',
        description: 'Transactional email delivery and logs',
      },
    ],
  },
];

export default function AdminObservabilityPage() {
  return (
    <AdminLayout>
      <div className="max-w-4xl mx-auto px-6 py-10">
        <h1 className="text-2xl font-semibold text-dark-100 mb-8">Observability & Infrastructure</h1>

        <div className="space-y-8">
          {groups.map((group) => (
            <section key={group.title}>
              <h2 className="text-xs font-semibold uppercase tracking-wider text-dark-500 mb-3">
                {group.title}
              </h2>
              <div className="border border-dark-700/50 rounded-lg divide-y divide-dark-700/50">
                {group.items.map((item) => (
                  <a
                    key={item.label}
                    href={item.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center justify-between px-4 py-3 hover:bg-dark-800/50 transition-colors group"
                  >
                    <div className="min-w-0">
                      <div className="text-sm font-medium text-dark-200 group-hover:text-accent transition-colors">
                        {item.label}
                      </div>
                      <div className="text-xs text-dark-500 mt-0.5 truncate">
                        {item.description}
                      </div>
                    </div>
                    <ExternalLink size={14} className="shrink-0 ml-4 text-dark-600 group-hover:text-accent transition-colors" />
                  </a>
                ))}
              </div>
            </section>
          ))}
        </div>
      </div>
    </AdminLayout>
  );
}
