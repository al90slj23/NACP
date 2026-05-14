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

import React, { useState, useEffect, useCallback } from 'react';
import {
  Table,
  DatePicker,
  Input,
  Button,
  Tag,
  Toast,
  Empty,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, timestamp2string } from '../../helpers';
import { formatQuotaToUSD } from './utils';

export { formatQuotaToUSD } from './utils';

const TracesPage = () => {
  const { t } = useTranslation();

  // 筛选状态
  const [dateRange, setDateRange] = useState(null);
  const [modelName, setModelName] = useState('');
  const [username, setUsername] = useState('');

  // 数据状态
  const [traces, setTraces] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // 展开行状态
  const [expandedRowKeys, setExpandedRowKeys] = useState([]);
  const [traceDetails, setTraceDetails] = useState({});
  const [detailLoading, setDetailLoading] = useState({});

  // 加载链路列表
  const loadTraces = useCallback(
    async (page = 1, size = 20) => {
      setLoading(true);
      try {
        let url = `/api/log/traces?p=${page}&page_size=${size}`;

        if (dateRange && dateRange.length === 2) {
          const startTimestamp = Math.floor(
            new Date(dateRange[0]).getTime() / 1000,
          );
          const endTimestamp = Math.floor(
            new Date(dateRange[1]).getTime() / 1000,
          );
          url += `&start_timestamp=${startTimestamp}&end_timestamp=${endTimestamp}`;
        }
        if (modelName) {
          url += `&model_name=${encodeURIComponent(modelName)}`;
        }
        if (username) {
          url += `&username=${encodeURIComponent(username)}`;
        }

        const res = await API.get(url);
        const { success, message, data } = res.data;
        if (success) {
          setTraces(data.items || []);
          setTotal(data.total || 0);
          setCurrentPage(data.page || page);
          setPageSize(data.page_size || size);
        } else {
          Toast.error(t('查询失败') + ': ' + message);
        }
      } catch (error) {
        Toast.error(t('查询失败'));
      }
      setLoading(false);
    },
    [dateRange, modelName, username, t],
  );

  // 加载链路详情
  const loadTraceDetail = useCallback(
    async (requestId) => {
      if (traceDetails[requestId]) {
        return;
      }
      setDetailLoading((prev) => ({ ...prev, [requestId]: true }));
      try {
        const res = await API.get(
          `/api/log/trace?request_id=${encodeURIComponent(requestId)}`,
        );
        const { success, message, data } = res.data;
        if (success) {
          setTraceDetails((prev) => ({ ...prev, [requestId]: data }));
        } else {
          Toast.error(message || t('查询失败'));
        }
      } catch (error) {
        Toast.error(t('查询失败'));
      }
      setDetailLoading((prev) => ({ ...prev, [requestId]: false }));
    },
    [traceDetails, t],
  );

  // 初始加载
  useEffect(() => {
    loadTraces(1, pageSize);
  }, []);

  // 查询
  const handleSearch = () => {
    setExpandedRowKeys([]);
    setTraceDetails({});
    loadTraces(1, pageSize);
  };

  // 重置
  const handleReset = () => {
    setDateRange(null);
    setModelName('');
    setUsername('');
    setExpandedRowKeys([]);
    setTraceDetails({});
    setCurrentPage(1);
    loadTraces(1, pageSize);
  };

  // 分页变化
  const handlePageChange = (page) => {
    setCurrentPage(page);
    setExpandedRowKeys([]);
    loadTraces(page, pageSize);
  };

  const handlePageSizeChange = (size) => {
    setPageSize(size);
    setCurrentPage(1);
    setExpandedRowKeys([]);
    loadTraces(1, size);
  };

  // 行展开/收起
  const handleExpand = (expanded, record) => {
    if (expanded) {
      setExpandedRowKeys((prev) => [...prev, record.request_id]);
      loadTraceDetail(record.request_id);
    } else {
      setExpandedRowKeys((prev) =>
        prev.filter((key) => key !== record.request_id),
      );
    }
  };

  // 渲染步骤状态
  const renderStepStatus = (step) => {
    switch (step.type) {
      case 51:
        return (
          <span style={{ color: 'red' }}>
            ❌ {t('已拦截')}
          </span>
        );
      case 52:
        return (
          <span style={{ color: 'red' }}>
            ❌ {t('客户端错误')}
          </span>
        );
      case 2:
        return (
          <span style={{ color: 'green' }}>
            ✅ {t('成功')} {formatQuotaToUSD(step.quota || 0)}
          </span>
        );
      default:
        return (
          <span style={{ color: 'orange' }}>
            ⚠️ {t('错误')}
          </span>
        );
    }
  };

  // 展开行渲染 — 链路详情时间线
  const expandedRowRender = (record) => {
    const requestId = record.request_id;
    const detail = traceDetails[requestId];
    const isLoading = detailLoading[requestId];

    if (isLoading) {
      return (
        <div style={{ padding: '12px 24px' }}>
          {t('加载中...')}
        </div>
      );
    }

    if (!detail || !detail.steps || detail.steps.length === 0) {
      return (
        <div style={{ padding: '12px 24px', color: 'var(--semi-color-text-2)' }}>
          {t('无链路数据')}
        </div>
      );
    }

    const steps = detail.steps;

    return (
      <div style={{ padding: '12px 24px' }}>
        {/* 请求摘要 */}
        <div style={{ marginBottom: 12, color: 'var(--semi-color-text-2)' }}>
          <Typography.Text type="tertiary">
            {t('请求时间')}: {timestamp2string(detail.created_at)} | {t('模型名称')}: {detail.model_name} | {t('用户名')}/{t('Token 名称')}: {detail.username}/{detail.token_name}
          </Typography.Text>
        </div>
        {/* 时间线步骤 */}
        <div style={{ fontFamily: 'monospace', fontSize: 13, lineHeight: '24px' }}>
          {steps.map((step, index) => {
            const isLast = index === steps.length - 1;
            const connector = isLast ? '└── ' : '├── ';
            return (
              <div key={step.id} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <span style={{ color: 'var(--semi-color-text-2)', whiteSpace: 'pre' }}>
                  {connector}
                </span>
                <span>
                  [{t('渠道ID')}: {step.channel_id}] {step.channel_name || '-'}
                </span>
                <span style={{ color: 'var(--semi-color-text-2)' }}>
                  | HTTP {step.status_code ?? '-'} | {step.use_time}s
                </span>
                <span>{renderStepStatus(step)}</span>
              </div>
            );
          })}
        </div>
      </div>
    );
  };

  // 表格列定义
  const columns = [
    {
      title: 'Request ID',
      dataIndex: 'request_id',
      key: 'request_id',
      width: 180,
      render: (text) => (
        <Typography.Text
          ellipsis={{ showTooltip: true }}
          style={{ maxWidth: 160 }}
        >
          {text}
        </Typography.Text>
      ),
    },
    {
      title: t('请求时间'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (value) => timestamp2string(value),
    },
    {
      title: t('模型名称'),
      dataIndex: 'model_name',
      key: 'model_name',
      width: 150,
    },
    {
      title: t('用户名') + '/' + t('Token 名称'),
      key: 'user_token',
      width: 160,
      render: (_, record) => (
        <span>
          {record.username || '-'}/{record.token_name || '-'}
        </span>
      ),
    },
    {
      title: t('最终结果'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status) =>
        status === 'success' ? (
          <Tag color="green">{t('成功')}</Tag>
        ) : (
          <Tag color="red">{t('失败')}</Tag>
        ),
    },
    {
      title: t('尝试渠道数'),
      dataIndex: 'channel_count',
      key: 'channel_count',
      width: 110,
    },
    {
      title: t('总耗时'),
      dataIndex: 'total_duration',
      key: 'total_duration',
      width: 100,
      render: (value) => `${value}s`,
    },
    {
      title: t('总消耗额度'),
      dataIndex: 'total_quota',
      key: 'total_quota',
      width: 140,
      render: (value) => formatQuotaToUSD(value || 0),
    },
  ];

  return (
    <div className="mt-[60px] px-2">
      {/* 筛选栏 */}
      <div
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: 12,
          alignItems: 'center',
          marginBottom: 16,
          padding: '16px',
          background: 'var(--semi-color-bg-1)',
          borderRadius: 8,
        }}
      >
        <DatePicker
          type="dateTimeRange"
          value={dateRange}
          onChange={(value) => setDateRange(value)}
          placeholder={[t('开始时间'), t('结束时间')]}
          style={{ width: 380 }}
        />
        <Input
          value={modelName}
          onChange={(value) => setModelName(value)}
          placeholder={t('模型名称')}
          maxLength={100}
          style={{ width: 160 }}
        />
        <Input
          value={username}
          onChange={(value) => setUsername(value)}
          placeholder={t('用户名')}
          maxLength={100}
          style={{ width: 160 }}
        />
        <Button theme="solid" type="primary" onClick={handleSearch}>
          {t('查询')}
        </Button>
        <Button theme="light" onClick={handleReset}>
          {t('重置')}
        </Button>
      </div>

      {/* 数据表格 */}
      <Table
        columns={columns}
        dataSource={traces}
        rowKey="request_id"
        loading={loading}
        expandedRowKeys={expandedRowKeys}
        onExpand={handleExpand}
        expandedRowRender={expandedRowRender}
        empty={
          <Empty
            description={t('搜索无结果')}
          />
        }
        pagination={{
          currentPage,
          pageSize,
          total,
          pageSizeOpts: [10, 20, 50, 100],
          showSizeChanger: true,
          onPageChange: handlePageChange,
          onPageSizeChange: handlePageSizeChange,
        }}
      />
    </div>
  );
};

export default TracesPage;
