/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useState, useEffect, useMemo, useRef } from 'react';
import { createPortal } from 'react-dom';
import {
  Spin,
  Tag,
  Empty,
  Typography,
  Tooltip,
  Toast,
  Button,
} from '@douyinfe/semi-ui';
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
  copy,
} from '../../../helpers';

const stepConfig = {
  2: { label: '2：正常消费成功', color: 'lime' },
  4: { label: '4：系统日志', color: 'purple' },
  5: { label: '5：普通错误', color: 'red' },
  21: { label: '21：容错重试成功', color: 'lime' },
  29: {
    label: '29：容错探测成功',
    color: 'white',
    style: {
      backgroundColor: 'rgba(var(--semi-teal-5), 0.14)',
      border: '1px solid rgba(var(--semi-teal-5), 0.22)',
      color: 'rgba(var(--semi-teal-8), 0.92)',
    },
  },
  51: { label: '51：容错重试已拦截', color: 'yellow' },
  52: { label: '52：容错重试客户端可见', color: 'red' },
  59: {
    label: '59：容错探测失败',
    color: 'white',
    style: {
      backgroundColor: 'rgba(var(--semi-red-5), 0.10)',
      border: '1px solid rgba(var(--semi-red-5), 0.18)',
      color: 'rgba(var(--semi-red-8), 0.86)',
    },
  },
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
  '72px 168px 190px 220px 150px 130px 190px 190px 128px 72px 72px 110px 130px 150px minmax(220px, 1fr)';

async function copyLogId(event, id, t) {
  event.stopPropagation();
  const text = String(id || '');
  if (!text) {
    return;
  }
  if (await copy(text)) {
    Toast.success(`${t('已复制')} Log ID: ${text}`);
  } else {
    Toast.error(t('复制失败'));
  }
}

function LogIdMarker({
  id,
  sequence,
  requestId,
  traceId,
  traceSeq,
  traceParentId,
  traceSiblingSeq,
  traceRole,
  prefix,
  t,
}) {
  const content = (
    <div style={{ lineHeight: 1.6 }}>
      <div>Log ID: {id || '-'}</div>
      <div>Trace ID: {traceId || requestId || '-'}</div>
      <div>Trace Seq: {traceSeq || sequence || '-'}</div>
      <div>Trace Role: {traceRole || '-'}</div>
      <div>Parent Log ID: {traceParentId || '-'}</div>
      <div>Sibling Seq: {traceSiblingSeq || '-'}</div>
      <div>Request ID: {requestId || '-'}</div>
      <div style={{ color: 'var(--semi-color-text-2)' }}>{t('点击复制')}</div>
    </div>
  );

  return (
    <Tooltip content={content} position='top'>
      <button
        type='button'
        onClick={(event) => copyLogId(event, id, t)}
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 4,
          width: 58,
          border: 'none',
          padding: 0,
          background: 'transparent',
          color: 'var(--semi-color-text-2)',
          fontFamily: 'monospace',
          cursor: 'copy',
        }}
      >
        <span>{prefix}</span>
        <Tag color='grey' size='small' shape='circle'>
          #{id || '-'}
        </Tag>
      </button>
    </Tooltip>
  );
}

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

function getStepTypeLabel(type, t) {
  const cfg = stepConfig[type];
  if (cfg?.label) {
    return t(cfg.label);
  }
  return `${type ?? 0}：${t('未知')}`;
}

function positiveNumber(value) {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
}

function formatStepTokenUsage(step, other, displayType, t) {
  const promptTokens = positiveNumber(step.prompt_tokens);
  const completionTokens = positiveNumber(step.completion_tokens);
  const cacheReadTokens = positiveNumber(other?.cache_tokens);
  const splitCacheCreation =
    positiveNumber(other?.cache_creation_tokens_5m) +
    positiveNumber(other?.cache_creation_tokens_1h);
  const cacheCreationTokens =
    splitCacheCreation > 0
      ? splitCacheCreation
      : positiveNumber(other?.cache_creation_tokens);
  const total =
    promptTokens + completionTokens + cacheReadTokens + cacheCreationTokens;
  const isProbe = displayType === 29 || displayType === 59;
  if (isProbe && total === 0 && other?.probe_usage_recorded !== true) {
    return `${t('上游未返回 usage')}（${t('无法计算 token')}）`;
  }
  const parts = [
    `${t('输入')} ${promptTokens}`,
    `${t('输出')} ${completionTokens}`,
  ];
  if (cacheReadTokens > 0) {
    parts.push(`${t('缓存读')} ${cacheReadTokens}`);
  }
  if (cacheCreationTokens > 0) {
    parts.push(`${t('缓存写')} ${cacheCreationTokens}`);
  }
  return `${t('合计')} ${total} tokens（${parts.join('，')}）`;
}

function formatStepCost(step, other, displayType, t) {
  const cost = renderQuota(step.quota || 0, 6);
  if (displayType === 29 || displayType === 59) {
    if (other?.probe_usage_recorded !== true && !positiveNumber(step.quota)) {
      return `${t('平台运营消耗')}：${t('未记录')}（${t('上游未返回 usage')}）`;
    }
    return `${t('平台运营消耗估算')}：${cost}（${t('轻量探测，不计入用户账单')}）`;
  }
  if (displayType === 51) {
    return `${t('用户扣款消耗')}：${cost}（${t('已拦截，未向用户返回')}）`;
  }
  if (displayType === 52) {
    return `${t('用户扣款消耗')}：${cost}（${t('失败收尾，客户端可见')}）`;
  }
  if (step.quota > 0) {
    return `${t('用户扣款消耗')}：${cost}`;
  }
  return `${t('用户扣款消耗')}：${cost}`;
}

function formatTimestampMs(timestampMs) {
  const date = new Date(timestampMs);
  const pad = (value, size = 2) => String(value).padStart(size, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(
    date.getDate(),
  )} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(
    date.getSeconds(),
  )}.${pad(date.getMilliseconds(), 3)}`;
}

function formatDuration(durationMs) {
  if (!Number.isFinite(durationMs) || durationMs <= 0) {
    return '0 秒';
  }
  if (durationMs < 1000) {
    return `${(durationMs / 1000).toFixed(3)} 秒（${Math.round(durationMs)} ms）`;
  }
  return `${(durationMs / 1000).toFixed(2)} 秒`;
}

function timestampMsFromTiming(timing, key) {
  const value = Number(timing?.[key]);
  return Number.isFinite(value) && value > 0 ? value : 0;
}

function formatTimePoint(label, timestampMs) {
  return `${label}：${timestampMs > 0 ? formatTimestampMs(timestampMs) : '-'}`;
}

function formatStepTimeRecord(step, other, displayType, t) {
  const timing = other?.timing || {};
  const latencyMs = positiveNumber(other?.admin_info?.latency_ms);
  const durationMs =
    latencyMs > 0 ? latencyMs : positiveNumber(step.use_time) * 1000;
  const fallbackEndMs = positiveNumber(step.created_at) * 1000;
  const fallbackStartMs =
    fallbackEndMs > 0 ? Math.max(0, fallbackEndMs - durationMs) : 0;
  const upstreamStartedAt =
    timestampMsFromTiming(timing, 'upstream_started_at_ms') || fallbackStartMs;
  const upstreamFinishedAt =
    timestampMsFromTiming(timing, 'upstream_finished_at_ms') || fallbackEndMs;
  const downstreamReceivedAt =
    timestampMsFromTiming(timing, 'downstream_received_at_ms') ||
    upstreamStartedAt;
  const downstreamFinishedAt =
    timestampMsFromTiming(timing, 'downstream_finished_at_ms') ||
    upstreamFinishedAt;
  const isFirstStep =
    displayType === 2 || step.trace_seq === 1 || step.sequence === 1;
  const isTerminalStep = [2, 21, 52].includes(displayType);
  const upstreamDuration =
    upstreamStartedAt > 0 && upstreamFinishedAt > upstreamStartedAt
      ? upstreamFinishedAt - upstreamStartedAt
      : durationMs;

  return (
    <div style={{ lineHeight: 1.65 }}>
      {isFirstStep && (
        <div>{formatTimePoint(t('下游↑'), downstreamReceivedAt)}</div>
      )}
      <div>{formatTimePoint(t('上游↑'), upstreamStartedAt)}</div>
      <div>{formatTimePoint(t('上游↓'), upstreamFinishedAt)}</div>
      {isTerminalStep && (
        <div>{formatTimePoint(t('下游↓'), downstreamFinishedAt)}</div>
      )}
      <div>
        {`${t('上游耗时')}：${t('共')} ${formatDuration(upstreamDuration)}`}
      </div>
    </div>
  );
}

function compactPath(path) {
  return Array.isArray(path) && path.length > 0 ? path.join(' → ') : '-';
}

function buildTraceChannelPath(steps) {
  const path = [];
  for (const step of steps || []) {
    if (!step?.channel_id) {
      continue;
    }
    if (path[path.length - 1] !== step.channel_id) {
      path.push(step.channel_id);
    }
  }
  return path;
}

function getStepRoleDescription(displayType, t) {
  switch (displayType) {
    case 2:
      return t('直接请求成功，成功响应已返回用户');
    case 21:
      return t('容错链路中的最终成功请求，成功响应已返回用户');
    case 51:
      return t('正式请求失败，错误已被 NACP 拦截，未返回给用户');
    case 52:
      return t('容错链路失败收尾，最终错误已返回用户');
    case 29:
      return t('轻量探测成功，不进入用户账单');
    case 59:
      return t('轻量探测失败，不进入用户账单');
    default:
      return t('日志记录');
  }
}

function getUserVisibleDescription(displayType, t) {
  if (displayType === 51 || displayType === 29 || displayType === 59) {
    return t('否');
  }
  if (displayType === 2 || displayType === 21 || displayType === 52) {
    return t('是');
  }
  return '-';
}

function getRetryDecisionDescription(displayType, traceContext, t) {
  if (displayType === 51) {
    const nextStep = traceContext?.nextStep;
    const nextDisplayType = getTraceStepDisplayType(nextStep);
    if (nextDisplayType === 52) {
      return t('本步错误已拦截；后续写入 52 作为最终客户端可见错误');
    }
    if (nextStep?.channel_id) {
      return `${t('继续尝试下一个渠道')}：${nextStep.channel_id}`;
    }
    return t('本步错误已拦截，等待后续容错决策');
  }
  if (displayType === 21 || displayType === 2) {
    return t('成功，停止继续重试');
  }
  if (displayType === 52) {
    return t('分组内无后续可尝试渠道，停止重试并返回错误');
  }
  return '-';
}

function buildErrorClassification(step, other, t) {
  const parts = [];
  const statusCode = other?.status_code ?? step.status_code;
  if (statusCode !== undefined && statusCode !== null) {
    parts.push(`HTTP ${statusCode}`);
  }
  if (other?.error_type) {
    parts.push(`${t('错误类型')}：${other.error_type}`);
  }
  if (other?.error_code) {
    parts.push(`${t('错误代码')}：${other.error_code}`);
  }
  return parts.length > 0 ? parts.join('；') : '';
}

function buildStepDetails(
  step,
  requestId,
  t,
  billingDisplayMode,
  displayType,
  traceContext = {},
) {
  const other = getLogOther(step.other);
  const details = [];

  details.push({
    key: t('渠道信息'),
    value: `${step.channel_id || '-'} - ${step.channel_name || '[未知]'}`,
  });
  if (requestId) {
    details.push({ key: t('Request ID'), value: requestId });
  }
  details.push({
    key: t('日志类型'),
    value: getStepTypeLabel(displayType, t),
  });
  details.push({
    key: t('链路位置'),
    value: `${t('第')} ${traceContext.index + 1} ${t('步')} / ${t('共')} ${
      traceContext.total
    } ${t('步')}`,
  });
  details.push({
    key: t('完整链路'),
    value: compactPath(traceContext.channelPath),
  });
  details.push({
    key: t('本步角色'),
    value: getStepRoleDescription(displayType, t),
  });
  details.push({
    key: t('用户可见'),
    value: getUserVisibleDescription(displayType, t),
  });
  details.push({
    key: t('重试决策'),
    value: getRetryDecisionDescription(displayType, traceContext, t),
  });
  const errorClassification = buildErrorClassification(step, other, t);
  if (errorClassification) {
    details.push({
      key: t('错误分类'),
      value: errorClassification,
    });
  }
  if (Array.isArray(other?.admin_info?.use_channel)) {
    details.push({
      key: t('已尝试渠道'),
      value: other.admin_info.use_channel.join(' → '),
    });
  }
  details.push({
    key: t('消耗 Token'),
    value: formatStepTokenUsage(step, other, displayType, t),
  });
  details.push({
    key: t('产生费用'),
    value: formatStepCost(step, other, displayType, t),
  });
  details.push({
    key: t('时间小计'),
    value: formatStepTimeRecord(step, other, displayType, t),
  });

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
    <Tag color={cfg.color} size='small' shape='circle' style={cfg.style}>
      {t(cfg.label)}
    </Tag>
  );
}

function StepDetailsPopoverContent({ details }) {
  return (
    <div
      style={{
        padding: 2,
      }}
    >
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 0,
          color: 'rgba(255, 255, 255, 0.9)',
          fontSize: 13,
        }}
      >
        {details.map((detail, index) => (
          <div
            key={`${detail.key}-${index}`}
            style={{
              display: 'grid',
              gridTemplateColumns: '140px minmax(0, 1fr)',
              columnGap: 14,
              padding: '7px 4px',
              borderBottom:
                index === details.length - 1
                  ? 'none'
                  : '1px solid rgba(255, 255, 255, 0.07)',
            }}
          >
            <div
              style={{
                color: 'rgba(255, 255, 255, 0.56)',
                whiteSpace: 'nowrap',
              }}
            >
              {detail.key}
            </div>
            <div
              style={{
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                overflowWrap: 'anywhere',
                lineHeight: 1.55,
              }}
            >
              {detail.value}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function formatDetailsForCopy(details) {
  return (details || [])
    .map((detail) => `${detail.key}\n${detail.value}`)
    .join('\n\n');
}

function getFloatingPanelPosition(mouse) {
  if (!mouse || typeof window === 'undefined') {
    return { left: 16, top: 16, maxHeight: 320 };
  }
  const panelWidth = Math.min(720, Math.max(360, window.innerWidth * 0.72));
  const viewportMargin = 12;
  const gap = 14;
  const minUsefulHeight = 260;
  let left = mouse.x + gap;
  const belowHeight = window.innerHeight - mouse.y - gap - viewportMargin;
  const aboveHeight = mouse.y - gap - viewportMargin;
  const placeBelow =
    belowHeight >= minUsefulHeight || belowHeight >= aboveHeight;
  const panelMaxHeight = Math.max(160, placeBelow ? belowHeight : aboveHeight);
  let top = placeBelow
    ? mouse.y + gap
    : Math.max(viewportMargin, mouse.y - gap - panelMaxHeight);
  if (left + panelWidth > window.innerWidth - viewportMargin) {
    left = mouse.x - panelWidth - gap;
  }
  left = Math.max(
    viewportMargin,
    Math.min(left, window.innerWidth - panelWidth - viewportMargin),
  );
  top = Math.max(
    viewportMargin,
    Math.min(top, window.innerHeight - panelMaxHeight - viewportMargin),
  );
  return { left, top, width: panelWidth, maxHeight: panelMaxHeight };
}

function FloatingStepDetails({
  hover,
  pinned = false,
  onMouseEnter,
  onMouseLeave,
  onClose,
  onCopyAll,
  t,
}) {
  const panelRef = useRef(null);
  const style = useMemo(
    () => getFloatingPanelPosition(hover?.mouse),
    [hover?.mouse],
  );
  useEffect(() => {
    if (!pinned || typeof document === 'undefined') {
      return undefined;
    }
    const handleOutsidePointerDown = (event) => {
      if (
        panelRef.current &&
        event.target instanceof Node &&
        !panelRef.current.contains(event.target)
      ) {
        onClose?.();
      }
    };
    document.addEventListener('pointerdown', handleOutsidePointerDown, true);
    return () => {
      document.removeEventListener(
        'pointerdown',
        handleOutsidePointerDown,
        true,
      );
    };
  }, [pinned, onClose]);

  if (!hover?.details?.length || typeof document === 'undefined') {
    return null;
  }
  return createPortal(
    <div
      ref={panelRef}
      onMouseEnter={pinned ? undefined : onMouseEnter}
      onMouseLeave={pinned ? undefined : onMouseLeave}
      onClick={(event) => event.stopPropagation()}
      style={{
        position: 'fixed',
        left: style.left,
        top: style.top,
        width: style.width,
        maxHeight: style.maxHeight,
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        zIndex: 2147483647,
        borderRadius: 8,
        border: '1px solid rgba(255, 255, 255, 0.18)',
        backgroundColor: '#07090d',
        backgroundImage:
          'linear-gradient(180deg, rgba(255,255,255,0.035), rgba(255,255,255,0))',
        color: '#f2f5f8',
        boxShadow:
          '0 26px 70px rgba(0, 0, 0, 0.86), 0 0 0 9999px rgba(0, 0, 0, 0.18)',
        isolation: 'isolate',
        contain: 'layout paint style',
        fontVariantNumeric: 'tabular-nums',
        pointerEvents: 'auto',
        userSelect: 'text',
      }}
    >
      {pinned && (
        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            flex: '0 0 auto',
            padding: '8px 10px 4px',
            borderBottom: '1px solid rgba(255, 255, 255, 0.06)',
            backgroundColor: '#07090d',
          }}
        >
          <Button
            size='small'
            theme='borderless'
            type='tertiary'
            onClick={onClose}
            style={{
              color: 'rgba(255, 255, 255, 0.72)',
              minWidth: 28,
              height: 28,
              padding: 0,
            }}
          >
            ×
          </Button>
        </div>
      )}
      <div
        style={{
          flex: '1 1 auto',
          minHeight: 0,
          overflow: 'auto',
          overscrollBehavior: 'contain',
          padding: pinned ? '10px 16px 12px' : '14px 16px',
        }}
      >
        <StepDetailsPopoverContent details={hover.details} />
      </div>
      {pinned && (
        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            flex: '0 0 auto',
            padding: '10px 12px',
            borderTop: '1px solid rgba(255, 255, 255, 0.08)',
            backgroundColor: '#07090d',
          }}
        >
          <Button size='small' theme='solid' type='primary' onClick={onCopyAll}>
            {t('复制全部')}
          </Button>
        </div>
      )}
    </div>,
    document.body,
  );
}

function getTraceStepDisplayType(step) {
  if (!step) {
    return 0;
  }
  switch (step.trace_role) {
    case 'consume':
      return step.trace_seq > 1 ? 21 : 2;
    case 'error_intercepted':
      return 51;
    case 'error_visible':
      return 52;
    case 'probe_success':
      return 29;
    case 'probe_failed':
      return 59;
    default:
      return step.type;
  }
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
  const [hoverDetails, setHoverDetails] = useState(null);
  const [pinnedDetails, setPinnedDetails] = useState(null);
  const hoverHideTimer = useRef(null);

  const clearHoverHideTimer = () => {
    if (hoverHideTimer.current) {
      window.clearTimeout(hoverHideTimer.current);
      hoverHideTimer.current = null;
    }
  };

  const showHoverDetails = (details, event) => {
    clearHoverHideTimer();
    setHoverDetails({
      details,
      mouse: { x: event.clientX, y: event.clientY },
    });
  };

  const pinStepDetails = (details, event) => {
    clearHoverHideTimer();
    setPinnedDetails({
      details,
      mouse: { x: event.clientX, y: event.clientY },
    });
  };

  const closePinnedDetails = () => {
    setPinnedDetails(null);
  };

  const copyPinnedDetails = async () => {
    const text = formatDetailsForCopy(pinnedDetails?.details);
    if (!text) {
      return;
    }
    if (await copy(text)) {
      Toast.success(t('复制成功'));
    } else {
      Toast.error(t('复制失败'));
    }
  };

  const scheduleHideHoverDetails = () => {
    clearHoverHideTimer();
    hoverHideTimer.current = window.setTimeout(() => {
      setHoverDetails(null);
      hoverHideTimer.current = null;
    }, 120);
  };

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

  useEffect(() => () => clearHoverHideTimer(), []);

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
        const displayType = getTraceStepDisplayType(step);
        const other = getLogOther(step.other);
        const traceContext = {
          index: idx,
          total: steps.length,
          nextStep: steps[idx + 1],
          channelPath: buildTraceChannelPath(steps),
        };
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
          displayType,
          traceContext,
        );

        return (
          <React.Fragment key={step.id}>
            <div
              onMouseEnter={(event) => showHoverDetails(details, event)}
              onMouseMove={(event) => showHoverDetails(details, event)}
              onMouseLeave={scheduleHideHoverDetails}
              onClick={(event) => {
                event.stopPropagation();
                pinStepDetails(details, event);
              }}
              style={{
                borderBottom: '1px solid var(--semi-color-border)',
                background:
                  idx % 2 === 0 ? 'transparent' : 'var(--semi-color-fill-0)',
                display: 'grid',
                gridTemplateColumns,
                minWidth: 2080,
                gap: 0,
                alignItems: 'center',
                padding: '8px',
                cursor: 'help',
              }}
            >
              <TraceCell
                style={{
                  overflow: 'visible',
                }}
              >
                <LogIdMarker
                  id={step.id}
                  sequence={step.sequence}
                  requestId={step.request_id || requestId}
                  traceId={step.trace_id}
                  traceSeq={step.trace_seq}
                  traceParentId={step.trace_parent_id}
                  traceSiblingSeq={step.trace_sibling_seq}
                  traceRole={step.trace_role}
                  prefix={prefix}
                  t={t}
                />
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
          </React.Fragment>
        );
      })}
      <FloatingStepDetails
        hover={pinnedDetails || hoverDetails}
        pinned={!!pinnedDetails}
        onMouseEnter={clearHoverHideTimer}
        onMouseLeave={scheduleHideHoverDetails}
        onClose={closePinnedDetails}
        onCopyAll={copyPinnedDetails}
        t={t}
      />
    </div>
  );
};

export default TraceExpandRender;
