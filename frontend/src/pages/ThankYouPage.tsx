import { Link } from 'react-router-dom';
import { Heart } from 'lucide-react';

export default function ThankYouPage() {
  return (
    <div className="max-w-2xl mx-auto px-4 py-16">
      <div className="bg-dark-900/30 border border-dark-800/50 rounded-xl p-8 text-center">
        <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-accent/10 border border-accent/20 mb-4">
          <Heart size={20} strokeWidth={1.5} className="text-accent" />
        </div>
        <h1 className="text-2xl font-semibold text-dark-100 mb-2">Thank you for supporting GovTrove</h1>
        <p className="text-sm text-dark-400 mb-6">
          Your contribution helps keep the service running. It means a lot.
        </p>
        <Link
          to="/"
          className="inline-flex items-center gap-1.5 px-6 py-3 text-sm font-semibold text-dark-950
                     rounded-lg bg-accent hover:bg-accent/90 transition-colors"
        >
          Back to search
        </Link>
      </div>
    </div>
  );
}
