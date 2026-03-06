import { X, Star, Bookmark, Bell } from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';

export type SignupPromptContext = 'bookmark' | 'save-search' | 'save-all';

const contextConfig: Record<SignupPromptContext, { icon: typeof Star; headline: string; subtext: string }> = {
  bookmark: {
    icon: Star,
    headline: 'Save this opportunity',
    subtext: 'Create a free account to bookmark opportunities and track them across devices.',
  },
  'save-search': {
    icon: Bookmark,
    headline: 'Save your search',
    subtext: 'Create a free account to save searches and return to them later.',
  },
  'save-all': {
    icon: Star,
    headline: 'Save these opportunities',
    subtext: 'Create a free account to bookmark opportunities and access them anytime.',
  },
};

interface SignupPromptModalProps {
  context: SignupPromptContext;
  onClose: () => void;
}

export default function SignupPromptModal({ context, onClose }: SignupPromptModalProps) {
  const { signUp, signIn } = useAppAuth();
  const config = contextConfig[context];
  const Icon = config.icon;

  return (
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div className="relative w-full max-w-sm mx-4 bg-dark-900 border border-dark-700/50 rounded-xl shadow-xl p-6">
        <button
          onClick={onClose}
          className="absolute top-3 right-3 p-1 text-dark-500 hover:text-dark-300 transition-colors"
        >
          <X size={16} />
        </button>

        <div className="flex flex-col items-center text-center">
          <div className="w-12 h-12 rounded-xl bg-accent/10 border border-accent/20 flex items-center justify-center mb-4">
            <Icon size={22} className="text-accent" strokeWidth={1.5} />
          </div>
          <h3 className="text-lg font-semibold text-dark-100 mb-2">{config.headline}</h3>
          <p className="text-sm text-dark-400 leading-relaxed mb-6">{config.subtext}</p>

          <button
            onClick={() => signUp()}
            className="w-full px-4 py-2.5 bg-accent hover:bg-accent-hover text-white font-medium rounded-lg
                       transition-all duration-200 text-sm hover:shadow-lg hover:shadow-accent/20"
          >
            Sign Up Free
          </button>

          <p className="mt-3 text-xs text-dark-500">
            Already have an account?{' '}
            <button
              onClick={() => signIn()}
              className="text-accent hover:text-accent/80 transition-colors"
            >
              Sign In
            </button>
          </p>

          <div className="mt-5 pt-4 border-t border-dark-800/50 w-full">
            <div className="flex items-center justify-center gap-4 text-[11px] text-dark-600">
              <span className="flex items-center gap-1"><Star size={10} /> Bookmark opps</span>
              <span className="flex items-center gap-1"><Bookmark size={10} /> Save searches</span>
              <span className="flex items-center gap-1"><Bell size={10} /> Alerts (Pro)</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
