import DesktopNav from './DesktopNav';
import MobileTabBar from './MobileTabBar';

export default function Navigation({ hideLogo }: { hideLogo: boolean }) {
  return (
    <>
      <DesktopNav hideLogo={hideLogo} />
      <MobileTabBar />
    </>
  );
}
