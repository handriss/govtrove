import { Link } from 'react-router-dom';
import { ArrowLeft, MapPinOff } from 'lucide-react';

export default function NotFoundPage() {
  return (
    <div className="min-h-screen relative">
      <div className="fixed inset-0 bg-gradient-to-br from-dark-900/30 via-transparent to-dark-950/50 pointer-events-none" />

      <div className="relative z-10 max-w-2xl mx-auto px-6 pt-10 pb-10">
        <div className="flex flex-col items-center text-center py-16">
          <div className="w-16 h-16 rounded-2xl bg-accent/10 border border-accent/20 flex items-center justify-center mb-6">
            <MapPinOff size={28} className="text-accent/60" strokeWidth={1.5} />
          </div>

          <h1 className="text-2xl font-semibold text-dark-100 mb-2">Page not found</h1>
          <p className="text-sm text-dark-400 max-w-md mb-8">
            The page you're looking for doesn't exist or has been moved.
          </p>

          <Link
            to="/"
            className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-dark-800/50 border border-dark-700/50 text-sm text-dark-200 hover:bg-dark-800 transition-colors"
          >
            <ArrowLeft size={16} />
            Back to Search
          </Link>
        </div>
      </div>
    </div>
  );
}
