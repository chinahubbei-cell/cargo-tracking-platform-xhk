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
          fontSize: 15,        // 基础字号 14→15
          fontSizeSM: 13,      // 小号字体 12→13  
          fontSizeLG: 17,      // 大号字体 16→17
          fontSizeXL: 21,      // 超大字体 20→21
          fontSizeHeading1: 38,
          fontSizeHeading2: 30,
          fontSizeHeading3: 24,
          fontSizeHeading4: 20,
          fontSizeHeading5: 17,
          lineHeight: 1.6,     // 行高 1.5→1.6
          borderRadius: 6,
        },
        components: {
          Table: {
            fontSize: 14,
            cellFontSize: 14,
          },
          Menu: {
            fontSize: 15,
            itemHeight: 44,
          },
          Button: {
            fontSize: 15,
            controlHeight: 36,
          },
          Input: {
            fontSize: 15,
          },
          Select: {
            fontSize: 15,
          },
          Modal: {
            fontSize: 15,
            titleFontSize: 18,
          },
          Card: {
            fontSize: 15,
          },
          Descriptions: {
            fontSize: 14,
          },
          Form: {
            fontSize: 15,
            labelFontSize: 15,
          },
          Tabs: {
            fontSize: 15,
          },
          Tag: {
            fontSize: 13,
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
