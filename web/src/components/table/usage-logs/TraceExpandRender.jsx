import React, { useState, useEffect } from 'react';
import { Spin, Tag, Empty, Space, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, renderQuota } from '../../../helpers';

const stepConfig = {
  51: { icon: '❌', label: '已拦截', color: 'red' },
  52: { icon: '❌', label: '客户端错误', color: 'red' },
  2: { icon: '✅', label: '成功', color: 'green' },
  21: { icon: '✅', label: '成功', color: 'green' },
  29: { icon: '🔍', label: '探测成功', color: 'blue' },
  59: { icon: '🔍', label: '探测失败', color: 'grey' },
};

const TraceExpandRender = ({ requestId }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [steps, setSteps] = useState([]);

  useEffect(() => {
    const fetchDetail = async () => {
      setLoading(true);
      try {
        const res = await API.get(
          `/api/log/trace?request_id=${encodeURIComponent(requestId)}`,
        );
        if (res.data.success) {
          setSteps(res.data.data.steps || []);
        }
      } catch (e) {
        console.error('Failed to fetch trace detail:', e);
      } finally {
        setLoading(false);
      }
    };
    if (requestId) {
      fetchDetail();
    } else {
      setLoading(false);
    }
  }, [requestId]);

  if (loading) {
    return (
      <div style={{ padding: '16px', textAlign: 'center' }}>
        <Spin />
      </div>
    );
  }

  if (steps.length === 0) {
    return <Empty description={t('无链路数据')} />;
  }

  return (
    <div style={{ padding: '8px 16px', fontFamily: 'monospace', fontSize: 13 }}>
      {steps.map((step, idx) => {
        const isLast = idx === steps.length - 1;
        const prefix = isLast ? '└── ' : '├── ';
        const cfg = stepConfig[step.type] || stepConfig[51];
        const tokenParts = [];
        if (step.prompt_tokens > 0) {
          tokenParts.push(`${t('输入')} ${step.prompt_tokens}`);
        }
        if (step.completion_tokens > 0) {
          tokenParts.push(`${t('输出')} ${step.completion_tokens}`);
        }

        return (
          <div
            key={step.id}
            style={{
              lineHeight: '26px',
              paddingBottom: 6,
              whiteSpace: 'normal',
              wordBreak: 'break-word',
            }}
          >
            <div>
              <span style={{ color: '#999' }}>{prefix}</span>
              <span>{cfg.icon} </span>
              <Tag color={cfg.color} size='small'>
                {t(cfg.label)}
              </Tag>
              <span style={{ marginLeft: 8 }}>
                CH#{step.channel_id} {step.channel_name}
              </span>
              {step.status_code != null && (
                <Tag color='light-blue' size='small' style={{ marginLeft: 8 }}>
                  HTTP {step.status_code}
                </Tag>
              )}
              <span style={{ marginLeft: 8, color: '#666' }}>
                {step.use_time}s
              </span>
              {step.quota > 0 && (
                <span style={{ marginLeft: 8, color: '#52c41a' }}>
                  {renderQuota(step.quota)}
                </span>
              )}
            </div>
            {(step.model_name ||
              step.token_name ||
              step.group ||
              step.ip ||
              tokenParts.length > 0 ||
              step.is_stream) && (
              <Space
                wrap
                spacing={6}
                style={{ marginLeft: 34, marginTop: 2, rowGap: 4 }}
              >
                {step.model_name && (
                  <Tag color='light-green' size='small'>
                    {step.model_name}
                  </Tag>
                )}
                {step.token_name && (
                  <Tag color='grey' size='small'>
                    {step.token_name}
                  </Tag>
                )}
                {step.group && (
                  <Tag color='light-blue' size='small'>
                    {step.group}
                  </Tag>
                )}
                {tokenParts.length > 0 && (
                  <Tag color='white' size='small'>
                    {tokenParts.join(' / ')}
                  </Tag>
                )}
                <Tag color={step.is_stream ? 'blue' : 'purple'} size='small'>
                  {step.is_stream ? t('流') : t('非流')}
                </Tag>
                {step.ip && (
                  <Tag color='orange' size='small'>
                    {step.ip}
                  </Tag>
                )}
              </Space>
            )}
            {step.content && (
              <Typography.Text
                type='tertiary'
                style={{
                  display: 'block',
                  marginLeft: 34,
                  maxWidth: 900,
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  lineHeight: 1.5,
                }}
              >
                {step.content}
              </Typography.Text>
            )}
          </div>
        );
      })}
    </div>
  );
};

export default TraceExpandRender;
