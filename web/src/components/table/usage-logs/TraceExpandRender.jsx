import React, { useState, useEffect } from 'react';
import { Spin, Tag, Empty, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  API,
  getLogOther,
  renderAudioModelPrice,
  renderClaudeLogContent,
  renderClaudeModelPrice,
  renderGroup,
  renderLogContent,
  renderModelPrice,
  renderQuota,
  renderTaskBillingProcess,
  renderTieredModelPrice,
  timestamp2string,
} from '../../../helpers';

const stepConfig = {
  2: { label: '2：正常消费成功', color: 'lime' },
  20: { label: '20：容错重试后成功', color: 'green' },
  21: { label: '21：容错重试成功', color: 'green' },
  29: { label: '29：容错探测成功', color: 'blue' },
  50: { label: '50：容错重试后失败', color: 'red' },
  51: { label: '51：容错重试已拦截', color: 'yellow' },
  52: { label: '52：容错重试客户端可见', color: 'red' },
  59: { label: '59：容错探测失败', color: 'grey' },
};

const colors = [
  'amber',
  'blue',
  'cyan',
  'green',
  'grey',
  'indigo',
  'light-blue',
  'lime',
  'orange',
  'pink',
  'purple',
  'red',
  'teal',
  'violet',
  'yellow',
];

const gridTemplateColumns =
  '38px 168px 190px 220px 150px 130px 190px 190px 128px 72px 72px 110px 130px 150px minmax(220px, 1fr)';

function requestConversionDisplayValue(conversionChain, t) {
  const chain = Array.isArray(conversionChain)
    ? conversionChain.filter(Boolean)
    : [];
  if (chain.length <= 1) {
    return t('原生格式');
  }
  return `${chain.join(' -> ')}`;
}

function buildStreamStatusValue(streamStatus, t) {
  if (!streamStatus) {
    return '';
  }
  const isOk = streamStatus.status === 'ok';
  let value =
    (isOk ? '✓ ' + t('正常') : '✗ ' + t('异常')) +
    ' (' +
    (streamStatus.end_reason || 'unknown') +
    ')';
  if (streamStatus.error_count > 0) {
    value += ` [${t('软错误')}: ${streamStatus.error_count}]`;
  }
  if (streamStatus.end_error) {
    value += ` - ${streamStatus.end_error}`;
  }
  return value;
}

function buildStepDetails(step, requestId, t, billingDisplayMode) {
  const other = getLogOther(step.other);
  const details = [];

  details.push({
    key: t('渠道信息'),
    value: `${step.channel_id || '-'} - ${step.channel_name || '[未知]'}`,
  });
  if (requestId) {
    details.push({ key: t('Request ID'), value: requestId });
  }

  if (other?.ws || other?.audio) {
    details.push({ key: t('语音输入'), value: other.audio_input });
    details.push({ key: t('语音输出'), value: other.audio_output });
    details.push({ key: t('文字输入'), value: other.text_input });
    details.push({ key: t('文字输出'), value: other.text_output });
  }
  if (other?.cache_tokens > 0) {
    details.push({ key: t('缓存 Tokens'), value: other.cache_tokens });
  }
  if (other?.cache_creation_tokens > 0) {
    details.push({
      key: t('缓存创建 Tokens'),
      value: other.cache_creation_tokens,
    });
  }

  if ((step.type === 2 || step.type === 21) && other) {
    if (other?.billing_mode !== 'tiered_expr') {
      details.push({
        key: t('日志详情'),
        value: other?.claude
          ? renderClaudeLogContent({
              ...other,
              displayMode: billingDisplayMode,
            })
          : renderLogContent({ ...other, displayMode: billingDisplayMode }),
      });
    }

    if (
      other?.is_model_mapped &&
      other?.upstream_model_name &&
      other?.upstream_model_name !== ''
    ) {
      details.push({ key: t('请求并计费模型'), value: step.model_name });
      details.push({ key: t('实际模型'), value: other.upstream_model_name });
    }

    const isViolationFeeLog =
      other?.violation_fee === true ||
      Boolean(other?.violation_fee_code) ||
      Boolean(other?.violation_fee_marker);
    if (!isViolationFeeLog && other?.billing_mode !== 'tiered_expr') {
      const logOpts = {
        ...other,
        prompt_tokens: step.prompt_tokens,
        completion_tokens: step.completion_tokens,
        displayMode: billingDisplayMode,
      };
      const isTaskLog = other?.is_task === true || other?.task_id != null;
      let content = '';
      if (isTaskLog && other?.model_price === -1) {
        content = renderTaskBillingProcess(other, step.content);
      } else if (other?.ws || other?.audio) {
        content = renderAudioModelPrice(logOpts);
      } else if (other?.claude) {
        content = renderClaudeModelPrice(logOpts);
      } else {
        content = renderModelPrice(logOpts);
      }
      details.push({ key: t('计费过程'), value: content });
    }
    if (other?.billing_mode === 'tiered_expr' && other?.expr_b64) {
      details.push({
        key: t('计费过程'),
        value: renderTieredModelPrice({
          ...other,
          prompt_tokens: step.prompt_tokens,
          completion_tokens: step.completion_tokens,
          displayMode: billingDisplayMode,
        }),
      });
    }
    if (other?.reasoning_effort) {
      details.push({
        key: t('Reasoning Effort'),
        value: other.reasoning_effort,
      });
    }
  }

  if (step.content) {
    details.push({ key: t('其他详情'), value: step.content });
  }
  if (other?.reject_reason) {
    details.push({ key: t('拦截原因'), value: other.reject_reason });
  }
  if (other?.reason) {
    details.push({ key: t('失败原因'), value: other.reason });
  }
  if (other?.request_path) {
    details.push({ key: t('请求路径'), value: other.request_path });
  }
  if (other?.stream_status) {
    details.push({
      key: t('流状态'),
      value: buildStreamStatusValue(other.stream_status, t),
    });
    if (
      Array.isArray(other.stream_status.errors) &&
      other.stream_status.errors.length > 0
    ) {
      details.push({
        key: t('流错误详情'),
        value: other.stream_status.errors.join('\n'),
      });
    }
  }
  if (other?.request_conversion) {
    details.push({
      key: t('请求转换'),
      value: requestConversionDisplayValue(other.request_conversion, t),
    });
  } else {
    details.push({ key: t('请求转换'), value: t('原生格式') });
  }
  details.push({
    key: t('计费模式'),
    value: other?.admin_info?.local_count_tokens
      ? t('本地计费')
      : t('上游返回'),
  });

  return details.filter(
    (item) =>
      item.value !== undefined && item.value !== null && item.value !== '',
  );
}

function TypeTag({ type, t }) {
  const cfg = stepConfig[type] || {
    label: `${type ?? 0}：${t('未知')}`,
    color: 'grey',
  };
  return (
    <Tag color={cfg.color} size='small' shape='circle'>
      {t(cfg.label)}
    </Tag>
  );
}

function getTraceStepDisplayType(type) {
  // The persisted consume log is still type=2 for billing/statistics. Inside a
  // retry summary trace it represents the final retry success, so display it as
  // 21 to avoid confusing it with a direct no-retry consume success.
  return type === 2 ? 21 : type;
}

function TraceCell({ children, className = '', style = {} }) {
  return (
    <div
      className={className}
      style={{
        minWidth: 0,
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        ...style,
      }}
    >
      {children}
    </div>
  );
}

const TraceExpandRender = ({ requestId, billingDisplayMode = 'price' }) => {
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
    <div
      style={{
        padding: '8px 12px',
        overflowX: 'auto',
        fontSize: 13,
      }}
    >
      <div
        style={{
          display: 'grid',
          gridTemplateColumns,
          minWidth: 2080,
          gap: 0,
          alignItems: 'center',
          padding: '6px 8px',
          color: 'var(--semi-color-text-2)',
          borderBottom: '1px solid var(--semi-color-border)',
          background: 'var(--semi-color-fill-0)',
          fontWeight: 600,
        }}
      >
        <TraceCell />
        <TraceCell>{t('时间')}</TraceCell>
        <TraceCell>{t('渠道')}</TraceCell>
        <TraceCell>{t('用户')}</TraceCell>
        <TraceCell>{t('令牌')}</TraceCell>
        <TraceCell>{t('分组')}</TraceCell>
        <TraceCell>{t('类型')}</TraceCell>
        <TraceCell>{t('模型')}</TraceCell>
        <TraceCell>{t('用时/首字')}</TraceCell>
        <TraceCell>{t('输入')}</TraceCell>
        <TraceCell>{t('输出')}</TraceCell>
        <TraceCell>{t('花费')}</TraceCell>
        <TraceCell>IP</TraceCell>
        <TraceCell>{t('重试')}</TraceCell>
        <TraceCell>{t('详情')}</TraceCell>
      </div>
      {steps.map((step, idx) => {
        const isLast = idx === steps.length - 1;
        const prefix = isLast ? '└── ' : '├── ';
        const displayType = getTraceStepDisplayType(step.type);
        const other = getLogOther(step.other);
        const retryPath = Array.isArray(other?.admin_info?.use_channel)
          ? other.admin_info.use_channel.join('->')
          : step.channel_id
            ? `${t('渠道')}：${step.channel_id}`
            : '';
        const details = buildStepDetails(
          step,
          requestId,
          t,
          billingDisplayMode,
        );

        return (
          <div
            key={step.id}
            style={{
              borderBottom: '1px solid var(--semi-color-border)',
              background:
                idx % 2 === 0 ? 'transparent' : 'var(--semi-color-fill-0)',
            }}
          >
            <div
              style={{
                display: 'grid',
                gridTemplateColumns,
                minWidth: 2080,
                gap: 0,
                alignItems: 'center',
                padding: '8px',
              }}
            >
              <TraceCell
                style={{
                  color: 'var(--semi-color-text-2)',
                  fontFamily: 'monospace',
                }}
              >
                {prefix}
              </TraceCell>
              <TraceCell>{timestamp2string(step.created_at)}</TraceCell>
              <TraceCell>
                {step.channel_id ? (
                  <span>
                    <Tag
                      color={colors[parseInt(step.channel_id) % colors.length]}
                      size='small'
                      shape='circle'
                    >
                      {step.channel_id}
                    </Tag>
                    {step.channel_name ? (
                      <span style={{ marginLeft: 6 }}>{step.channel_name}</span>
                    ) : null}
                  </span>
                ) : (
                  '-'
                )}
              </TraceCell>
              <TraceCell>{step.username || '-'}</TraceCell>
              <TraceCell>
                {step.token_name ? (
                  <Tag color='grey' size='small' shape='circle'>
                    {step.token_name}
                  </Tag>
                ) : (
                  '-'
                )}
              </TraceCell>
              <TraceCell>
                {step.group ? renderGroup(step.group) : '-'}
              </TraceCell>
              <TraceCell>
                <TypeTag type={displayType} t={t} />
                {step.status_code != null && (
                  <Tag
                    color='light-blue'
                    size='small'
                    style={{ marginLeft: 6 }}
                  >
                    HTTP {step.status_code}
                  </Tag>
                )}
              </TraceCell>
              <TraceCell>{step.model_name || '-'}</TraceCell>
              <TraceCell>
                <Tag
                  color={step.use_time < 10 ? 'green' : 'orange'}
                  size='small'
                  shape='circle'
                >
                  {step.use_time} s
                </Tag>
                <Tag
                  color={step.is_stream ? 'blue' : 'purple'}
                  size='small'
                  shape='circle'
                  style={{ marginLeft: 4 }}
                >
                  {step.is_stream ? t('流') : t('非流')}
                </Tag>
              </TraceCell>
              <TraceCell>{step.prompt_tokens || 0}</TraceCell>
              <TraceCell>{step.completion_tokens || 0}</TraceCell>
              <TraceCell>
                {step.quota > 0 ? renderQuota(step.quota, 6) : '-'}
              </TraceCell>
              <TraceCell>{step.ip || '-'}</TraceCell>
              <TraceCell>{retryPath || '-'}</TraceCell>
              <TraceCell>
                <Typography.Text
                  ellipsis={{
                    showTooltip: {
                      type: 'popover',
                      opts: { style: { width: 300 } },
                    },
                  }}
                  style={{ maxWidth: 210 }}
                >
                  {step.content || other?.request_path || '-'}
                </Typography.Text>
              </TraceCell>
            </div>
            {details.length > 0 && (
              <div
                style={{
                  minWidth: 2080,
                  marginLeft: 46,
                  padding: '0 8px 10px 8px',
                }}
              >
                <div
                  style={{
                    borderLeft: '2px solid var(--semi-color-border)',
                    paddingLeft: 12,
                    display: 'grid',
                    gridTemplateColumns: '132px minmax(0, 1fr)',
                    rowGap: 6,
                    columnGap: 10,
                    color: 'var(--semi-color-text-1)',
                  }}
                >
                  {details.map((detail) => (
                    <React.Fragment key={detail.key}>
                      <div
                        style={{
                          color: 'var(--semi-color-text-2)',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {detail.key}
                      </div>
                      <div
                        style={{
                          whiteSpace: 'pre-wrap',
                          wordBreak: 'break-word',
                          lineHeight: 1.55,
                        }}
                      >
                        {detail.value}
                      </div>
                    </React.Fragment>
                  ))}
                </div>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
};

export default TraceExpandRender;
