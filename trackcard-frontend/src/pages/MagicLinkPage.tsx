import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Card, Typography, Button, Result, Spin, Alert, message, Upload, Input } from 'antd';
import { CameraOutlined, CheckCircleOutlined, EnvironmentOutlined, UploadOutlined } from '@ant-design/icons';
import type { UploadFile } from 'antd/es/upload/interface';
import api from '../api/client';

const { TextArea } = Input;
const { Title, Text } = Typography;

// Magic Link 操作页面数据接口
interface MagicLinkPageData {
  shipment_id: string;
  action_type: string;
  action_title: string;
  description: string;
  need_photo: boolean;
  need_gps: boolean;
  target_name: string;
}

// 位置信息接口
interface LocationInfo {
  latitude: number;
  longitude: number;
  accuracy?: number;
}

const MagicLinkPage: React.FC = () => {
  const { token } = useParams<{ token: string }>();
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [valid, setValid] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pageData, setPageData] = useState<MagicLinkPageData | null>(null);
  const [submitted, setSubmitted] = useState(false);

  // 表单数据
  const [remarks, setRemarks] = useState('');
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [location, setLocation] = useState<LocationInfo | null>(null);
  const [gettingLocation, setGettingLocation] = useState(false);

  // 加载页面数据
  useEffect(() => {
    if (!token) {
      setError('链接无效');
      setLoading(false);
      return;
    }

    const fetchPageData = async () => {
      try {
        const response = await api.get(`/api/m/${token}`);
        if (response.data.valid) {
          setValid(true);
          setPageData(response.data.data);
          // 如果需要GPS，自动获取位置
          if (response.data.data.need_gps) {
            getLocation();
          }
        } else {
          setError(response.data.error || '链接无效或已过期');
        }
      } catch (err: any) {
        setError(err.response?.data?.error || '加载失败，请稍后重试');
      } finally {
        setLoading(false);
      }
    };

    fetchPageData();
  }, [token]);

  // 获取位置信息
  const getLocation = () => {
    if (!navigator.geolocation) {
      message.warning('您的浏览器不支持定位功能');
      return;
    }

    setGettingLocation(true);
    navigator.geolocation.getCurrentPosition(
      (position) => {
        setLocation({
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          accuracy: position.coords.accuracy,
        });
        setGettingLocation(false);
        message.success('位置获取成功');
      },
      (error) => {
        setGettingLocation(false);
        switch (error.code) {
          case error.PERMISSION_DENIED:
            message.error('请允许访问您的位置信息');
            break;
          case error.POSITION_UNAVAILABLE:
            message.error('无法获取位置信息');
            break;
          case error.TIMEOUT:
            message.error('获取位置超时');
            break;
          default:
            message.error('获取位置失败');
        }
      },
      {
        enableHighAccuracy: true,
        timeout: 10000,
        maximumAge: 0,
      }
    );
  };

  // 提交操作
  const handleSubmit = async () => {
    if (!token || !pageData) return;

    // 验证必填项
    if (pageData.need_photo && fileList.length === 0) {
      message.error('请上传照片');
      return;
    }

    if (pageData.need_gps && !location) {
      message.error('请允许获取您的位置信息');
      return;
    }

    setSubmitting(true);
    try {
      // 准备提交数据
      const submitData: any = {
        remarks,
      };

      if (location) {
        submitData.latitude = location.latitude;
        submitData.longitude = location.longitude;
      }

      // 如果有上传的文件，添加URL（实际项目中应先上传文件获取URL）
      if (fileList.length > 0) {
        submitData.photo_urls = fileList
          .filter(f => f.status === 'done')
          .map(f => f.response?.url || f.url);
      }

      const response = await api.post(`/api/m/${token}/submit`, submitData);

      if (response.data.success) {
        setSubmitted(true);
        message.success(response.data.message || '提交成功！');
      } else {
        message.error(response.data.error || '提交失败');
      }
    } catch (err: any) {
      message.error(err.response?.data?.error || '提交失败，请稍后重试');
    } finally {
      setSubmitting(false);
    }
  };

  // 渲染加载状态
  if (loading) {
    return (
      <div className="magic-link-container loading">
        <Spin size="large" tip="正在加载..." />
      </div>
    );
  }

  // 渲染错误状态
  if (!valid || error) {
    return (
      <div className="magic-link-container">
        <Result
          status="error"
          title="链接无效"
          subTitle={error || '该链接已过期或已被使用，请联系发送方获取新链接'}
          extra={
            <Button type="primary" onClick={() => window.close()}>
              关闭页面
            </Button>
          }
        />
      </div>
    );
  }

  // 渲染提交成功状态
  if (submitted) {
    return (
      <div className="magic-link-container">
        <Result
          status="success"
          title="操作成功"
          subTitle="感谢您的配合！信息已成功提交"
          extra={
            <Button type="primary" onClick={() => window.close()}>
              关闭页面
            </Button>
          }
        />
      </div>
    );
  }

  // 渲染操作表单
  return (
    <div className="magic-link-container">
      <Card className="magic-link-card">
        {/* 头部 */}
        <div className="magic-link-header">
          <Title level={4}>{pageData?.action_title}</Title>
          <Text type="secondary">{pageData?.description}</Text>
        </div>

        {/* 运单信息 */}
        <Alert
          message={`运单号：${pageData?.shipment_id}`}
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />

        {/* 欢迎语 */}
        {pageData?.target_name && (
          <div className="welcome-text">
            <Text>您好，{pageData.target_name}！</Text>
          </div>
        )}

        {/* 照片上传区域 */}
        {pageData?.need_photo && (
          <div className="form-section">
            <Title level={5}>
              <CameraOutlined /> 上传照片
            </Title>
            <Upload
              listType="picture-card"
              fileList={fileList}
              onChange={({ fileList }) => setFileList(fileList)}
              beforeUpload={() => false} // 不自动上传，由提交时处理
              accept="image/*"
              maxCount={5}
            >
              {fileList.length < 5 && (
                <div>
                  <UploadOutlined />
                  <div style={{ marginTop: 8 }}>点击上传</div>
                </div>
              )}
            </Upload>
            <Text type="secondary">支持拍照或从相册选择，最多5张</Text>
          </div>
        )}

        {/* 位置信息 */}
        {pageData?.need_gps && (
          <div className="form-section">
            <Title level={5}>
              <EnvironmentOutlined /> 位置信息
            </Title>
            {location ? (
              <Alert
                message={`已获取位置 (精度: ${location.accuracy?.toFixed(0) || '未知'}米)`}
                type="success"
                showIcon
                icon={<CheckCircleOutlined />}
              />
            ) : (
              <Button
                type="dashed"
                onClick={getLocation}
                loading={gettingLocation}
                icon={<EnvironmentOutlined />}
                block
              >
                {gettingLocation ? '正在获取位置...' : '点击获取当前位置'}
              </Button>
            )}
          </div>
        )}

        {/* 备注 */}
        <div className="form-section">
          <Title level={5}>备注说明（选填）</Title>
          <TextArea
            value={remarks}
            onChange={(e) => setRemarks(e.target.value)}
            placeholder="如有特殊情况，请在此说明"
            rows={3}
            maxLength={500}
            showCount
          />
        </div>

        {/* 提交按钮 */}
        <Button
          type="primary"
          size="large"
          block
          onClick={handleSubmit}
          loading={submitting}
          icon={<CheckCircleOutlined />}
          className="submit-button"
        >
          确认提交
        </Button>
      </Card>

      {/* 底部品牌 */}
      <div className="magic-link-footer">
        <Text type="secondary">Powered by TrackCard物流平台</Text>
      </div>

      <style>{`
        .magic-link-container {
          min-height: 100vh;
          background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
          padding: 16px;
          display: flex;
          flex-direction: column;
          align-items: center;
        }
        
        .magic-link-container.loading {
          justify-content: center;
        }
        
        .magic-link-card {
          width: 100%;
          max-width: 500px;
          border-radius: 16px;
          box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
        }
        
        .magic-link-header {
          text-align: center;
          margin-bottom: 20px;
          padding-bottom: 16px;
          border-bottom: 1px solid #f0f0f0;
        }
        
        .magic-link-header h4 {
          margin-bottom: 8px;
          color: #1a1a1a;
        }
        
        .welcome-text {
          text-align: center;
          margin-bottom: 20px;
          padding: 12px;
          background: #f5f5f5;
          border-radius: 8px;
        }
        
        .form-section {
          margin-bottom: 24px;
        }
        
        .form-section h5 {
          margin-bottom: 12px;
          color: #333;
        }
        
        .submit-button {
          margin-top: 16px;
          height: 48px;
          font-size: 16px;
          border-radius: 8px;
          background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
          border: none;
        }
        
        .submit-button:hover {
          background: linear-gradient(135deg, #5a6fd6 0%, #6a4190 100%);
        }
        
        .magic-link-footer {
          margin-top: 24px;
          text-align: center;
        }
        
        .magic-link-footer .ant-typography {
          color: rgba(255, 255, 255, 0.7);
          font-size: 12px;
        }
        
        /* 移动端优化 */
        @media (max-width: 480px) {
          .magic-link-container {
            padding: 12px;
          }
          
          .magic-link-card {
            border-radius: 12px;
          }
        }
      `}</style>
    </div>
  );
};

export default MagicLinkPage;
