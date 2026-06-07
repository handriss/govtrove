import { Outlet } from 'react-router-dom';
import { Gift } from 'lucide-react';
import { useAppAuth } from '../contexts/AuthContext';
import { SidebarProvider, useSidebar } from '../contexts/SidebarContext';
import { PRO_FEATURES_FREE_FOR_ALL } from '../lib/billing';
import AppSidebar from './AppSidebar';
import BottomTabBar from './BottomTabBar';
import AuthButton from './AuthButton';
import Footer from './Footer';

function PromoBanner() {
  const { isAuthenticated, signUp, govtroveUser } = useAppAuth();
  const hasPromo = sessionStorage.getItem('govtrove_promo_code');
  const hasPro = PRO_FEATURES_FREE_FOR_ALL || govtroveUser?.plan === 'pro' || govtroveUser?.free_forever || (govtroveUser?.gift_expires_at && new Date(govtroveUser.gift_expires_at) > new Date());

  if (!hasPromo || hasPro) return null;

  if (!isAuthenticated) {
    return (
      <div className="bg-accent/10 border-b border-accent/30 px-4 py-3">
        <div className="max-w-4xl mx-auto flex items-center justify-center gap-3 flex-wrap">
          <Gift size={16} className="text-accent shrink-0" />
          <p className="text-sm text-dark-200">
            You have a special offer! Sign up to claim your discount on GovTrove Pro.
          </p>
          <button
            onClick={() => signUp()}
            className="px-4 py-1.5 text-xs font-semibold rounded-lg bg-accent text-dark-950 hover:bg-accent/90 transition-colors"
          >
            Sign Up Now
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-accent/10 border-b border-accent/30 px-4 py-3">
      <div className="max-w-4xl mx-auto flex items-center justify-center gap-3 flex-wrap">
        <Gift size={16} className="text-accent shrink-0" />
        <p className="text-sm text-dark-200">
          You have a special offer for GovTrove Pro!
        </p>
        <a
          href={`/profile?promo=${hasPromo}`}
          className="px-4 py-1.5 text-xs font-semibold rounded-lg bg-accent text-dark-950 hover:bg-accent/90 transition-colors"
        >
          Claim Offer
        </a>
      </div>
    </div>
  );
}

function AppLayoutInner() {
  const { isAuthenticated } = useAppAuth();
  const { collapsed } = useSidebar();
  const marginLeft = isAuthenticated ? (collapsed ? 'md:ml-14' : 'md:ml-56') : '';

  return (
    <>
      <PromoBanner />
      {isAuthenticated && (
        <>
          <AppSidebar />
          <BottomTabBar />
        </>
      )}
      <div className="fixed top-4 right-6 z-20">
        <AuthButton />
      </div>
      <main className={`${marginLeft} transition-all duration-200 min-h-screen ${isAuthenticated ? 'pb-20 md:pb-0' : ''}`}>
        <Outlet />
        <Footer />
      </main>
    </>
  );
}

export default function AppLayout() {
  return (
    <SidebarProvider>
      <AppLayoutInner />
    </SidebarProvider>
  );
}
