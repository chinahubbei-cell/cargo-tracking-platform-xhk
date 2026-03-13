import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, App as AntApp } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import MainLayout from './components/Layout/MainLayout';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Devices from './pages/Devices';
import Shipments from './pages/Shipments';
import CargoTracking from './pages/CargoTracking';
import RoutePlanning from './pages/RoutePlanning';
import GlobalPorts from './pages/GlobalPorts';
import GlobalAirports from './pages/GlobalAirports';
import Alerts from './pages/Alerts';
import Users from './pages/Users';
import Organizations from './pages/Organizations';
import Partners from './pages/Partners';
import Customers from './pages/Customers';
import Rates from './pages/Rates';
import Settings from './pages/Settings';
import MagicLinkPage from './pages/MagicLinkPage'; // Phase 8: B端轻量化页面

import { useAuthStore } from './store/authStore';

import 'antd/dist/reset.css';

// 私有路由保护组件
const PrivateRoute = ({ children }: { children: React.ReactNode }) => {
  const { token } = useAuthStore();

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};

function App() {
  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        token: {
          fontSize: 17.5,
          fontSizeSM: 15,
          fontSizeLG: 20,
          fontSizeXL: 25,
          fontSizeHeading1: 35,
          fontSizeHeading2: 30,
          fontSizeHeading3: 25,
          fontSizeHeading4: 22.5,
          fontSizeHeading5: 20,
          lineHeight: 1.5,
          borderRadius: 6,
        },
        components: {
          Table: {
            fontSize: 16,
            cellFontSize: 16,
            cellPaddingBlock: 10,
            cellPaddingInline: 15,
          },
          Menu: {
            fontSize: 17.5,
            itemHeight: 52,
          },
          Button: {
            fontSize: 17.5,
            controlHeight: 42,
          },
          Input: {
            fontSize: 17.5,
          },
          Select: {
            fontSize: 17.5,
          },
          Modal: {
            fontSize: 17.5,
            titleFontSize: 21,
          },
          Card: {
            fontSize: 17.5,
          },
          Descriptions: {
            fontSize: 16,
          },
          Form: {
            fontSize: 17.5,
            labelFontSize: 17.5,
          },
          Tabs: {
            fontSize: 17.5,
          },
          Tag: {
            fontSize: 15,
          },
        }
      }}
    >
      <AntApp>

        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<Login />} />
            {/* Phase 8: Magic Link 公开页面（无需认证） */}
            <Route path="/m/:token" element={<MagicLinkPage />} />
            <Route
              path="/"
              element={
                <PrivateRoute>
                  <MainLayout />
                </PrivateRoute>
              }
            >
              <Route index element={<Navigate to="/dashboard" replace />} />
              <Route path="dashboard" element={<Dashboard />} />

              {/* 货运业务 */}
              <Route path="business/shipments" element={<Shipments />} />
              <Route path="business/tracking" element={<CargoTracking />} />
              <Route path="business/route-planning" element={<RoutePlanning />} />
              <Route path="business/alerts" element={<Alerts />} />

              {/* 基础数据 */}
              <Route path="resources/ports" element={<GlobalPorts />} />
              <Route path="resources/airports" element={<GlobalAirports />} />
              <Route path="resources/partners" element={<Partners />} />
              <Route path="resources/customers" element={<Customers />} />

              {/* 设备管理 */}
              <Route path="devices/list" element={<Devices />} />

              {/* 组织管理 */}
              <Route path="organization/users" element={<Users />} />
              <Route path="organization/departments" element={<Organizations />} />

              {/* 商务管理 */}
              <Route path="business-mgmt/rates" element={<Rates />} />


              {/* 系统设置 */}
              <Route path="settings/system" element={<Settings />} />
              <Route path="settings/profile" element={<Settings />} />
            </Route>
            <Route path="*" element={<Navigate to="/dashboard" replace />} />
          </Routes>
        </BrowserRouter>
      </AntApp>
    </ConfigProvider>
  );
}

export default App;
