import { Outlet } from 'react-router-dom';
import { useAppAuth } from '../contexts/AuthContext';
import { SidebarProvider, useSidebar } from '../contexts/SidebarContext';
import AppSidebar from './AppSidebar';
import BottomTabBar from './BottomTabBar';
import AuthButton from './AuthButton';
import Footer from './Footer';

function AppLayoutInner() {
  const { isAuthenticated } = useAppAuth();
  const { collapsed } = useSidebar();
  const marginLeft = isAuthenticated ? (collapsed ? 'md:ml-14' : 'md:ml-56') : '';

  return (
    <>
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
