import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconPlus, IconRefresh, IconSearch } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import CardPro from '../../common/ui/CardPro';
import {
  API,
  showError,
  showInfo,
  showSuccess,
  timestamp2string,
} from '../../../helpers';
import { createCardProPagination } from '../../../helpers/utils';
import { useIsMobile } from '../../../hooks/common/useIsMobile';

const { Text } = Typography;

const STATUS_OPTIONS = [
  { label: '启用', value: 1 },
  { label: '停用', value: 2 },
];

const RANGE_OPTIONS = [
  { label: '1小时', value: 1 },
  { label: '6小时', value: 6 },
  { label: '24小时', value: 24 },
  { label: '7天', value: 168 },
  { label: '30天', value: 720 },
];

const DASHBOARD_TABS = [
  { label: '综合', value: 'overview' },
  { label: '模型&性能', value: 'model_performance' },
  { label: '用户&消耗', value: 'user_consumption' },
  { label: '流水&成本', value: 'cost_flow' },
  { label: '单位&渠道', value: 'unit_channel' },
];

const TAB_META = {
  overview: {
    title: '综合态势',
    desc: '集中查看请求质量、容错重试、用户、模型、渠道和单位账号统计源的整体状态。',
    scope: '全局',
    focus: ['请求成功率', '用户可见错误', 'SFT 容错结果', '统计源健康'],
  },
  model_performance: {
    title: '模型&性能',
    desc: '按模型维度检查请求量、成功率、流式占比、平均耗时、Token 和额度消耗。',
    scope: '模型',
    focus: ['模型排行', '错误模型', '耗时异常', '流式占比'],
  },
  user_consumption: {
    title: '用户&消耗',
    desc: '按用户维度检查消耗、活跃用户、错误影响和额度使用，方便定位大户和异常调用。',
    scope: '用户',
    focus: ['用户排行', '活跃用户', 'Token 消耗', '用户侧错误'],
  },
  cost_flow: {
    title: '流水&成本',
    desc: '把今日用户侧成功消费流水和上游账号余额快照差值放在一起，估算今日利润。',
    scope: '今日',
    focus: ['平台收入', '上游成本', '预计利润', '快照覆盖'],
  },
  unit_channel: {
    title: '单位&渠道',
    desc: '检查单位账号统计源、渠道健康、渠道调用排行和上游账号余额/额度采集状态。',
    scope: '单位/渠道',
    focus: ['单位余额', '渠道健康', '渠道排行', '采集能力'],
  },
};

const TREND_COLORS = {
  blue: '#4f8cff',
  green: '#3fb950',
  red: '#f87171',
  orange: '#f59e0b',
  purple: '#8b5cf6',
};

const formatMoney = (value, unit = 'USD') => {
  const num = Number(value || 0);
  return `${unit || 'USD'} ${num.toFixed(6)}`;
};

const formatQuota = (value) => Number(value || 0).toLocaleString();

const clampPercent = (value) => Math.max(0, Math.min(100, Number(value || 0)));

const formatPercentPerm = (value) => `${(Number(value || 0) / 10).toFixed(1)}%`;

const formatSeconds = (value) => `${Number(value || 0).toFixed(0)}s`;

const emptyOverview = {
  generated_at: 0,
  range_start: 0,
  range_end: 0,
  range_hours: 24,
  unit_monitors: {
    total: 0,
    enabled: 0,
    ok: 0,
    error: 0,
    pending: 0,
    current_balance: 0,
    used_amount: 0,
    token_count: 0,
    model_count: 0,
    group_count: 0,
    last_checked_time: 0,
  },
  users: {
    total: 0,
    enabled: 0,
    disabled: 0,
    admin: 0,
    active_24h: 0,
    active_users: 0,
    quota: 0,
    used_quota: 0,
    request_count: 0,
  },
  channels: {
    total: 0,
    enabled: 0,
    manually_disabled: 0,
    auto_disabled: 0,
    healthy: 0,
    degraded: 0,
    unhealthy: 0,
    used_quota: 0,
  },
  sft: {
    success_summary: 0,
    failed_summary: 0,
    intercepted: 0,
    client_visible: 0,
    probe_success: 0,
    probe_failed: 0,
    direct_consume: 0,
    retry_consume: 0,
    legacy_error: 0,
    success_rate_perm: 0,
  },
  traffic: {
    request_count: 0,
    success_count: 0,
    error_count: 0,
    user_visible_errors: 0,
    intercepted_errors: 0,
    probe_count: 0,
    stream_count: 0,
    stream_rate_perm: 0,
    tokens: 0,
    quota: 0,
    average_use_time: 0,
    success_rate_perm: 0,
  },
  cost: {
    day_start: 0,
    day_end: 0,
    platform_revenue_quota: 0,
    platform_revenue_usd: 0,
    upstream_cost_usd: 0,
    estimated_profit_usd: 0,
    gross_margin_perm: 0,
    monitor_count: 0,
    baseline_monitor_count: 0,
    missing_baseline_count: 0,
    snapshot_count: 0,
    estimated: true,
    note: '',
  },
  health: {
    score_perm: 1000,
    level: 'healthy',
    traffic_success_perm: 1000,
    unit_health_perm: 1000,
    channel_health_perm: 1000,
    cost_coverage_perm: 1000,
    notes: [],
  },
  trend: [],
  models: [],
  top_users: [],
  top_channels: [],
};

const parseRawJSON = (value) => {
  if (!value) return {};
  try {
    return JSON.parse(value);
  } catch (error) {
    return {};
  }
};

const formatDisplayValue = (value) => {
  if (value === undefined || value === null || value === '') return '-';
  if (typeof value === 'boolean') return value ? '是' : '否';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
};

const MonitorStatusTag = ({ value }) => {
  if (value === 'ok') {
    return (
      <Tag color='green' shape='circle'>
        正常
      </Tag>
    );
  }
  if (value === 'error') {
    return (
      <Tag color='red' shape='circle'>
        异常
      </Tag>
    );
  }
  return (
    <Tag color='grey' shape='circle'>
      待检测
    </Tag>
  );
};

const RANK_ENDPOINTS = {
  model_performance: 'models',
  user_consumption: 'users',
  unit_channel: 'channels',
};

const COST_DIMENSION_OPTIONS = [
  { label: '单位/账号', value: 'unit' },
  { label: '渠道', value: 'channel' },
  { label: '模型', value: 'model' },
  { label: '用户', value: 'user' },
];

const STATS_DIMENSION_OPTIONS = [
  { label: '分组', value: 'group' },
  { label: '令牌', value: 'token' },
  { label: '端点', value: 'endpoint' },
  { label: 'IP', value: 'ip' },
];

const emptyRankData = {
  items: [],
  total: 0,
  page: 1,
  page_size: 10,
};

const isNotFoundError = (error) => error?.response?.status === 404;

const NacpStatsTable = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [monitors, setMonitors] = useState([]);
  const [loading, setLoading] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [keyword, setKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [modalVisible, setModalVisible] = useState(false);
  const [modalLoading, setModalLoading] = useState(false);
  const [units, setUnits] = useState([]);
  const [unitOptions, setUnitOptions] = useState([]);
  const [accountOptions, setAccountOptions] = useState([]);
  const [selectedUnitId, setSelectedUnitId] = useState(undefined);
  const [checkingIds, setCheckingIds] = useState({});
  const [detailRecord, setDetailRecord] = useState(null);
  const [overviewLoading, setOverviewLoading] = useState(false);
  const [overview, setOverview] = useState(emptyOverview);
  const [rangeHours, setRangeHours] = useState(24);
  const [activeDashboard, setActiveDashboard] = useState('overview');
  const [rankLoading, setRankLoading] = useState(false);
  const [rankPage, setRankPage] = useState(1);
  const [rankPageSize, setRankPageSize] = useState(10);
  const [rankSort, setRankSort] = useState('requests');
  const [costDimension, setCostDimension] = useState('unit');
  const [costSort, setCostSort] = useState('revenue');
  const [costRankLoading, setCostRankLoading] = useState(false);
  const [costRankPage, setCostRankPage] = useState(1);
  const [costRankPageSize, setCostRankPageSize] = useState(10);
  const [costRankData, setCostRankData] = useState(emptyRankData);
  const [statsDimension, setStatsDimension] = useState('group');
  const [dimensionSort, setDimensionSort] = useState('requests');
  const [dimensionRankLoading, setDimensionRankLoading] = useState(false);
  const [dimensionRankPage, setDimensionRankPage] = useState(1);
  const [dimensionRankPageSize, setDimensionRankPageSize] = useState(10);
  const [dimensionRankData, setDimensionRankData] = useState(emptyRankData);
  const [coverageLoading, setCoverageLoading] = useState(false);
  const [coverageSort, setCoverageSort] = useState('coverage');
  const [coveragePage, setCoveragePage] = useState(1);
  const [coveragePageSize, setCoveragePageSize] = useState(10);
  const [coverageData, setCoverageData] = useState(emptyRankData);
  const [rankData, setRankData] = useState({
    models: emptyRankData,
    users: emptyRankData,
    channels: emptyRankData,
  });

  const rangeLabel = useMemo(
    () =>
      RANGE_OPTIONS.find((item) => item.value === rangeHours)?.label ||
      `${rangeHours}${t('小时')}`,
    [rangeHours, t],
  );

  const activeTabMeta = TAB_META[activeDashboard] || TAB_META.overview;

  const queryOverview = useCallback(async () => {
    setOverviewLoading(true);
    try {
      const res = await API.get('/api/nacp_stats/overview', {
        params: { hours: rangeHours },
      });
      const { success, data, message } = res.data;
      if (!success) {
        showError(message || t('获取NACP统计概览失败'));
        return;
      }
      setOverview({ ...emptyOverview, ...(data || {}) });
    } catch (error) {
      showError(t('获取NACP统计概览失败'));
    } finally {
      setOverviewLoading(false);
    }
  }, [rangeHours, t]);

  const activeRankKey = RANK_ENDPOINTS[activeDashboard];

  const queryRank = useCallback(
    async (key = activeRankKey, page = rankPage, size = rankPageSize) => {
      if (!key) return;
      setRankLoading(true);
      try {
        const res = await API.get(`/api/nacp_stats/${key}`, {
          params: {
            hours: rangeHours,
            p: page,
            page_size: size,
            sort: rankSort,
          },
        });
        const { success, data, message } = res.data;
        if (!success) {
          showError(message || t('获取NACP排行失败'));
          return;
        }
        setRankData((state) => ({
          ...state,
          [key]: { ...emptyRankData, ...(data || {}) },
        }));
        setRankPage(data?.page || page);
        setRankPageSize(data?.page_size || size);
      } catch (error) {
        showError(t('获取NACP排行失败'));
      } finally {
        setRankLoading(false);
      }
    },
    [activeRankKey, rangeHours, rankPage, rankPageSize, rankSort, t],
  );

  const queryCostRank = useCallback(
    async (page = costRankPage, size = costRankPageSize) => {
      setCostRankLoading(true);
      try {
        const res = await API.get('/api/nacp_stats/costs', {
          params: {
            dimension: costDimension,
            p: page,
            page_size: size,
            sort: costSort,
          },
          skipErrorHandler: true,
        });
        const { success, data, message } = res.data;
        if (!success) {
          showError(message || t('获取流水成本排行失败'));
          return;
        }
        setCostRankData({ ...emptyRankData, ...(data || {}) });
        setCostRankPage(data?.page || page);
        setCostRankPageSize(data?.page_size || size);
      } catch (error) {
        if (isNotFoundError(error)) {
          setCostRankData({ ...emptyRankData, page, page_size: size });
          setCostRankPage(page);
          setCostRankPageSize(size);
          return;
        }
        showError(t('获取流水成本排行失败'));
      } finally {
        setCostRankLoading(false);
      }
    },
    [costDimension, costRankPage, costRankPageSize, costSort, t],
  );

  const queryDimensionRank = useCallback(
    async (page = dimensionRankPage, size = dimensionRankPageSize) => {
      if (activeDashboard === 'cost_flow') return;
      setDimensionRankLoading(true);
      try {
        const res = await API.get('/api/nacp_stats/dimensions', {
          params: {
            hours: rangeHours,
            dimension: statsDimension,
            p: page,
            page_size: size,
            sort: dimensionSort,
          },
          skipErrorHandler: true,
        });
        const { success, data, message } = res.data;
        if (!success) {
          showError(message || t('获取维度排行失败'));
          return;
        }
        setDimensionRankData({ ...emptyRankData, ...(data || {}) });
        setDimensionRankPage(data?.page || page);
        setDimensionRankPageSize(data?.page_size || size);
      } catch (error) {
        if (isNotFoundError(error)) {
          setDimensionRankData({ ...emptyRankData, page, page_size: size });
          setDimensionRankPage(page);
          setDimensionRankPageSize(size);
          return;
        }
        showError(t('获取维度排行失败'));
      } finally {
        setDimensionRankLoading(false);
      }
    },
    [
      activeDashboard,
      dimensionRankPage,
      dimensionRankPageSize,
      dimensionSort,
      rangeHours,
      statsDimension,
      t,
    ],
  );

  const queryModelCoverage = useCallback(
    async (page = coveragePage, size = coveragePageSize) => {
      if (activeDashboard !== 'model_performance') return;
      setCoverageLoading(true);
      try {
        const res = await API.get('/api/nacp_stats/model_coverage', {
          params: {
            hours: rangeHours,
            p: page,
            page_size: size,
            sort: coverageSort,
          },
          skipErrorHandler: true,
        });
        const { success, data, message } = res.data;
        if (!success) {
          showError(message || t('获取模型覆盖矩阵失败'));
          return;
        }
        setCoverageData({ ...emptyRankData, ...(data || {}) });
        setCoveragePage(data?.page || page);
        setCoveragePageSize(data?.page_size || size);
      } catch (error) {
        if (isNotFoundError(error)) {
          setCoverageData({ ...emptyRankData, page, page_size: size });
          setCoveragePage(page);
          setCoveragePageSize(size);
          return;
        }
        showError(t('获取模型覆盖矩阵失败'));
      } finally {
        setCoverageLoading(false);
      }
    },
    [
      activeDashboard,
      coveragePage,
      coveragePageSize,
      coverageSort,
      rangeHours,
      t,
    ],
  );

  const queryMonitors = useCallback(
    async (page = activePage, size = pageSize) => {
      setLoading(true);
      try {
        const res = await API.get('/api/unit_monitor/', {
          params: {
            p: page,
            page_size: size,
            keyword,
            status: statusFilter,
          },
        });
        const { success, message, data } = res.data;
        if (!success) {
          showError(message);
          return;
        }
        setMonitors(data.items || []);
        setTotal(data.total || 0);
        setActivePage(data.page || page);
        setPageSize(data.page_size || size);
      } catch (error) {
        showError(t('获取NACP统计列表失败'));
      } finally {
        setLoading(false);
      }
    },
    [activePage, keyword, pageSize, statusFilter, t],
  );

  const loadUnits = async () => {
    try {
      const res = await API.get('/api/unit/', {
        params: { p: 1, page_size: 200, status: 1 },
      });
      const { success, data, message } = res.data;
      if (!success) {
        showError(message || t('获取单位列表失败'));
        return;
      }
      const items = data.items || [];
      setUnits(items);
      setUnitOptions(
        items.map((unit) => ({
          label: `${unit.name || `#${unit.id}`} · ${unit.type || 'newapi'}`,
          value: unit.id,
        })),
      );
    } catch (error) {
      showError(t('获取单位列表失败'));
    }
  };

  const loadAccounts = async (unitId) => {
    const parsedUnitId = Number(unitId || 0);
    setAccountOptions([]);
    if (!parsedUnitId) return;
    try {
      const res = await API.get(`/api/unit/${parsedUnitId}/accounts`);
      const { success, data, message } = res.data;
      if (!success) {
        showError(message || t('获取单位账号失败'));
        return;
      }
      setAccountOptions(
        (data || []).map((account) => ({
          label: `${account.account || account.account_id || `#${account.id}`} · ID: ${account.account_id || '-'}`,
          value: account.id,
        })),
      );
    } catch (error) {
      showError(t('获取单位账号失败'));
    }
  };

  useEffect(() => {
    queryMonitors(1, pageSize);
  }, [keyword, statusFilter]);

  useEffect(() => {
    queryMonitors(activePage, pageSize);
  }, []);

  useEffect(() => {
    queryOverview();
  }, [queryOverview]);

  useEffect(() => {
    setRankPage(1);
  }, [activeDashboard, rangeHours, rankSort]);

  useEffect(() => {
    if (!activeRankKey) return;
    queryRank(activeRankKey, rankPage, rankPageSize);
  }, [activeRankKey, queryRank, rankPage, rankPageSize]);

  useEffect(() => {
    setCostRankPage(1);
  }, [costDimension, costSort]);

  useEffect(() => {
    if (activeDashboard !== 'cost_flow') return;
    queryCostRank(costRankPage, costRankPageSize);
  }, [activeDashboard, queryCostRank, costRankPage, costRankPageSize]);

  useEffect(() => {
    const defaults = {
      overview: 'group',
      model_performance: 'endpoint',
      user_consumption: 'token',
      unit_channel: 'group',
    };
    if (defaults[activeDashboard]) {
      setStatsDimension(defaults[activeDashboard]);
      setDimensionRankPage(1);
    }
  }, [activeDashboard]);

  useEffect(() => {
    setDimensionRankPage(1);
  }, [rangeHours, statsDimension, dimensionSort]);

  useEffect(() => {
    if (activeDashboard === 'cost_flow') return;
    queryDimensionRank(dimensionRankPage, dimensionRankPageSize);
  }, [activeDashboard, queryDimensionRank, dimensionRankPage, dimensionRankPageSize]);

  useEffect(() => {
    if (activeDashboard !== 'model_performance') return;
    queryModelCoverage(coveragePage, coveragePageSize);
  }, [activeDashboard, queryModelCoverage, coveragePage, coveragePageSize]);

  const openCreateModal = () => {
    setSelectedUnitId(undefined);
    setAccountOptions([]);
    setModalVisible(true);
    loadUnits().then();
  };

  const submitMonitor = async (values) => {
    if (!values.unit_id || !values.unit_account_id) {
      showInfo(t('请选择单位和账号'));
      return;
    }
    setModalLoading(true);
    try {
      const unit = units.find((item) => item.id === Number(values.unit_id));
      const payload = {
        unit_id: Number(values.unit_id),
        unit_account_id: Number(values.unit_account_id),
        name: values.name || unit?.name || '',
      };
      const res = await API.post('/api/unit_monitor/', payload);
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || t('创建NACP统计失败'));
        return;
      }
      showSuccess(t('NACP统计已创建'));
      setModalVisible(false);
      queryMonitors(1, pageSize);
      if (data?.id) {
        checkMonitor(data.id);
      }
    } catch (error) {
      showError(t('创建NACP统计失败'));
    } finally {
      setModalLoading(false);
    }
  };

  const checkMonitor = async (id) => {
    setCheckingIds((state) => ({ ...state, [id]: true }));
    try {
      const res = await API.post(`/api/unit_monitor/${id}/check`);
      const { success, message } = res.data;
      if (!success) {
        showError(message || t('刷新NACP统计失败'));
        return;
      }
      showSuccess(t('NACP统计已刷新'));
      queryMonitors(activePage, pageSize);
      queryOverview();
    } catch (error) {
      showError(t('刷新NACP统计失败'));
    } finally {
      setCheckingIds((state) => ({ ...state, [id]: false }));
    }
  };

  const checkAll = async () => {
    setLoading(true);
    try {
      const res = await API.post('/api/unit_monitor/check_all');
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || t('刷新全部NACP统计失败'));
        return;
      }
      showSuccess(
        `${t('刷新完成')}：${t('成功')} ${data.ok || 0}，${t('失败')} ${data.failed || 0}`,
      );
      queryMonitors(activePage, pageSize);
      queryOverview();
    } catch (error) {
      showError(t('刷新全部NACP统计失败'));
    } finally {
      setLoading(false);
    }
  };

  const updateMonitorStatus = async (record, status) => {
    try {
      const res = await API.put('/api/unit_monitor/', {
        id: record.id,
        name: record.name,
        status,
      });
      const { success, message } = res.data;
      if (!success) {
        showError(message || t('更新NACP统计失败'));
        return;
      }
      showSuccess(t('NACP统计已更新'));
      queryMonitors(activePage, pageSize);
      queryOverview();
    } catch (error) {
      showError(t('更新NACP统计失败'));
    }
  };

  const deleteMonitor = async (record) => {
    try {
      const res = await API.delete(`/api/unit_monitor/${record.id}`);
      const { success, message } = res.data;
      if (!success) {
        showError(message || t('删除NACP统计失败'));
        return;
      }
      showSuccess(t('NACP统计已删除'));
      queryMonitors(activePage, pageSize);
      queryOverview();
    } catch (error) {
      showError(t('删除NACP统计失败'));
    }
  };

  const formatRawJSON = (value) => {
    if (!value) return '{}';
    try {
      return JSON.stringify(JSON.parse(value), null, 2);
    } catch (error) {
      return String(value);
    }
  };

  const renderInfoGrid = (items) => (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: '150px minmax(0, 1fr)',
        gap: '10px 14px',
        fontSize: 14,
      }}
    >
      {items.map((item) => (
        <React.Fragment key={item.label}>
          <Text type='tertiary'>{item.label}</Text>
          <Text>{formatDisplayValue(item.value)}</Text>
        </React.Fragment>
      ))}
    </div>
  );
  
  const renderDetailSummary = (record) => {
    const raw = parseRawJSON(record.raw_json);
    const self = raw.user_self || {};
    const collector = raw.collector || {};
    const collectedFields = Array.isArray(raw.collected_fields)
      ? raw.collected_fields.join('、')
      : '';

    return (
      <div className='space-y-4'>
        <div>
          <Text strong>{t('统计对象')}</Text>
          <div style={{ marginTop: 10 }}>
            {renderInfoGrid([
              { label: t('单位'), value: record.unit_name },
              { label: t('平台类型'), value: record.unit_type },
              { label: t('API 地址'), value: record.unit_api_url },
              { label: t('账号'), value: record.account },
              { label: t('账号 ID'), value: record.account_id },
              { label: t('统计状态'), value: record.platform_status },
              {
                label: t('最后检查'),
                value: record.last_checked_time
                  ? timestamp2string(record.last_checked_time)
                  : '-',
              },
            ])}
          </div>
        </div>
        <div>
          <Text strong>{t('余额与使用')}</Text>
          <div style={{ marginTop: 10 }}>
            {renderInfoGrid([
              {
                label: t('当前余额'),
                value: formatMoney(record.current_balance, record.balance_unit),
              },
              {
                label: t('已用额度'),
                value: formatMoney(record.used_amount, record.balance_unit),
              },
              { label: t('请求次数'), value: self.request_count },
              { label: t('原始余额字段 quota'), value: self.quota },
              { label: t('原始已用字段 used_quota'), value: self.used_quota },
            ])}
          </div>
        </div>
        <div>
          <Text strong>{t('上游账号')}</Text>
          <div style={{ marginTop: 10 }}>
            {renderInfoGrid([
              { label: t('上游用户 ID'), value: record.upstream_user_id },
              { label: t('用户名'), value: record.upstream_username },
              { label: t('显示名'), value: self.display_name },
              { label: t('分组'), value: record.upstream_group },
              { label: t('角色'), value: self.role },
              { label: t('状态'), value: self.status },
            ])}
          </div>
        </div>
        <div>
          <Text strong>{t('采集能力')}</Text>
          <div style={{ marginTop: 10 }}>
            {renderInfoGrid([
              { label: t('采集器'), value: collector.adapter },
              { label: t('采集字段'), value: collectedFields },
              { label: t('令牌数量'), value: record.token_count },
              { label: t('模型数量'), value: record.model_count },
              { label: t('分组数量'), value: record.group_count },
              { label: t('错误信息'), value: record.error_message },
            ])}
          </div>
        </div>
      </div>
    );
  };

  const copyDetail = async () => {
    if (!detailRecord) return;
    const text = formatRawJSON(detailRecord.raw_json);
    try {
      await navigator.clipboard.writeText(text);
      showSuccess(t('已复制'));
    } catch (error) {
      showInfo(t('复制失败，请手动选择复制'));
    }
  };

  const cardStyle = {
    border: '1px solid var(--semi-color-border)',
    background: 'var(--semi-color-bg-1)',
    borderRadius: 8,
    padding: 14,
    minHeight: 116,
  };

  const panelStyle = {
    border: '1px solid var(--semi-color-border)',
    background: 'var(--semi-color-fill-0)',
    borderRadius: 8,
    padding: 12,
    minHeight: 104,
  };

  const renderDashboardHeader = () => (
    <div
      style={{
        border: '1px solid var(--semi-color-border)',
        background:
          'linear-gradient(180deg, var(--semi-color-bg-1), var(--semi-color-fill-0))',
        borderRadius: 8,
        padding: isMobile ? 12 : 14,
        marginBottom: 12,
      }}
    >
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-start',
          gap: 12,
          flexWrap: 'wrap',
        }}
      >
        <div style={{ minWidth: 0 }}>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
            <Text strong style={{ fontSize: 18 }}>
              {t(activeTabMeta.title)}
            </Text>
            <Tag color='blue' size='small'>
              {t(activeTabMeta.scope)}
            </Tag>
            <Tag color={overviewLoading ? 'grey' : 'green'} size='small'>
              {overviewLoading ? t('刷新中') : t('实时')}
            </Tag>
          </div>
          <Text type='tertiary' size='small' style={{ display: 'block', marginTop: 6 }}>
            {t(activeTabMeta.desc)}
          </Text>
        </div>
        <Space wrap spacing={6}>
          <Tag color='cyan' size='small'>
            {t('范围')} {rangeLabel}
          </Tag>
          <Tag color='purple' size='small'>
            {t('生成')} {overview.generated_at ? timestamp2string(overview.generated_at) : '-'}
          </Tag>
        </Space>
      </div>
      <div style={{ marginTop: 12 }}>
        <Space wrap spacing={6}>
          {(activeTabMeta.focus || []).map((item) => (
            <Tag key={item} color='grey' size='small'>
              {t(item)}
            </Tag>
          ))}
        </Space>
      </div>
    </div>
  );

  const renderCardsGrid = (cards) => (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: isMobile
          ? '1fr'
          : 'repeat(4, minmax(180px, 1fr))',
        gap: 12,
        marginBottom: 14,
      }}
    >
      {cards.map((card) => (
        <div
          key={card.title}
          style={{
            ...cardStyle,
            borderTop: `3px solid ${TREND_COLORS[card.accent || 'blue'] || TREND_COLORS.blue}`,
          }}
        >
          <div className='flex justify-between items-start gap-2'>
            <Text type='secondary'>{card.title}</Text>
            {overviewLoading ? (
              <Tag color='grey' size='small'>
                {t('刷新中')}
              </Tag>
            ) : null}
          </div>
          <div style={{ fontSize: 24, fontWeight: 700, marginTop: 10 }}>
            {card.value}
          </div>
          <Text type='tertiary' size='small'>
            {card.meta}
          </Text>
          {typeof card.progress === 'number' && (
            <div
              style={{
                height: 6,
                borderRadius: 999,
                background: 'var(--semi-color-fill-1)',
                overflow: 'hidden',
                marginTop: 10,
              }}
            >
              <div
                style={{
                  height: '100%',
                  width: `${clampPercent(card.progress)}%`,
                  background: TREND_COLORS[card.accent || 'blue'] || TREND_COLORS.blue,
                }}
              />
            </div>
          )}
          <div style={{ marginTop: 10 }}>
            <Space wrap spacing={6}>
              {(card.tags || []).map((tag) => (
                <Tag key={tag.label} color={tag.color} size='small'>
                  {tag.label}
                </Tag>
              ))}
            </Space>
          </div>
        </div>
      ))}
    </div>
  );

  const renderPanelsGrid = (panels) => (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: isMobile
          ? '1fr'
          : 'repeat(4, minmax(180px, 1fr))',
        gap: 12,
        marginBottom: 14,
      }}
    >
      {panels.map((panel) => (
        <div key={panel.title} style={panelStyle}>
          <div className='flex justify-between items-center gap-2'>
            <Text strong>{panel.title}</Text>
            <Tag color={panel.color || 'green'} size='small'>
              {panel.status || t('已接入')}
            </Tag>
          </div>
          <div style={{ marginTop: 10 }}>
            {(panel.lines || []).map((line) => (
              <div key={line}>
                <Text type='tertiary' size='small'>
                  {line}
                </Text>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );

  const renderTrendPanel = (title, valueKey, color, formatter = formatQuota) => {
    const trend = overview.trend || [];
    const maxValue = Math.max(
      1,
      ...trend.map((point) => Number(point[valueKey] || 0)),
    );
    return (
      <div style={{ ...panelStyle, minHeight: 156 }}>
        <div className='flex justify-between items-center gap-2'>
          <Text strong>{title}</Text>
          <Tag color={color} size='small'>
            {rangeLabel}
          </Tag>
        </div>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: `repeat(${Math.max(trend.length, 1)}, minmax(4px, 1fr))`,
            gap: 3,
            height: 84,
            alignItems: 'end',
            marginTop: 14,
          }}
        >
          {(trend.length ? trend : [{ bucket_start: 0 }]).map((point) => {
            const value = Number(point[valueKey] || 0);
            return (
              <div
                key={`${valueKey}-${point.bucket_start}`}
                title={`${point.bucket_start ? timestamp2string(point.bucket_start) : '-'} · ${formatter(value)}`}
                style={{
                  minHeight: 2,
                  height: `${Math.max(2, (value / maxValue) * 84)}px`,
                  borderRadius: 3,
                  background: TREND_COLORS[color] || color,
                  opacity: value > 0 ? 0.9 : 0.18,
                }}
              />
            );
          })}
        </div>
        <Text type='tertiary' size='small'>
          {t('按当前范围自动分桶')}
        </Text>
      </div>
    );
  };

  const dashboardCards = useMemo(
    () => {
      const unitMonitors = overview.unit_monitors || emptyOverview.unit_monitors;
      const users = overview.users || emptyOverview.users;
      const channels = overview.channels || emptyOverview.channels;
      const sft = overview.sft || emptyOverview.sft;
      const traffic = overview.traffic || emptyOverview.traffic;
      const cost = overview.cost || emptyOverview.cost;
      const health = overview.health || emptyOverview.health;
      const commonCards = [
        {
          title: t('运营健康分'),
          value: formatPercentPerm(health.score_perm),
          meta: `${t('请求')} ${formatPercentPerm(health.traffic_success_perm)} · ${t('统计源')} ${formatPercentPerm(health.unit_health_perm)}`,
          accent:
            health.level === 'critical'
              ? 'red'
              : health.level === 'warning'
                ? 'orange'
                : 'green',
          progress: Number(health.score_perm || 0) / 10,
          tags: [
            {
              color:
                health.level === 'critical'
                  ? 'red'
                  : health.level === 'warning'
                    ? 'orange'
                    : 'green',
              label:
                health.level === 'critical'
                  ? t('危险')
                  : health.level === 'warning'
                    ? t('关注')
                    : t('健康'),
            },
            { color: 'blue', label: `${t('渠道')} ${formatPercentPerm(health.channel_health_perm)}` },
            { color: 'cyan', label: `${t('成本覆盖')} ${formatPercentPerm(health.cost_coverage_perm)}` },
          ],
        },
        {
          title: t('请求质量'),
          value: traffic.request_count,
          meta: `${t('成功率')} ${formatPercentPerm(traffic.success_rate_perm)} · ${t('平均耗时')} ${formatSeconds(traffic.average_use_time)}`,
          accent: 'blue',
          progress: Number(traffic.success_rate_perm || 0) / 10,
          tags: [
            { color: 'green', label: `${t('成功')} ${traffic.success_count}` },
            { color: 'red', label: `${t('错误')} ${traffic.error_count}` },
            { color: 'blue', label: `${t('流式')} ${formatPercentPerm(traffic.stream_rate_perm)}` },
          ],
        },
        {
          title: t('消耗统计'),
          value: formatQuota(traffic.tokens),
          meta: `${t('额度')} ${formatQuota(traffic.quota)} · ${t('探测')} ${traffic.probe_count}`,
          accent: 'green',
          tags: [
            { color: 'orange', label: `${t('拦截')} ${traffic.intercepted_errors}` },
            { color: 'red', label: `${t('用户可见错误')} ${traffic.user_visible_errors}` },
          ],
        },
        {
          title: t('今日流水成本'),
          value: formatMoney(cost.estimated_profit_usd),
          meta: `${t('平台收入')} ${formatMoney(cost.platform_revenue_usd)} · ${t('上游成本')} ${formatMoney(cost.upstream_cost_usd)}`,
          accent: cost.estimated_profit_usd >= 0 ? 'green' : 'red',
          progress: cost.platform_revenue_usd > 0 ? Number(cost.gross_margin_perm || 0) / 10 : 0,
          tags: [
            {
              color: cost.estimated_profit_usd >= 0 ? 'green' : 'red',
              label: `${t('毛利率')} ${formatPercentPerm(cost.gross_margin_perm)}`,
            },
            {
              color: cost.estimated ? 'orange' : 'blue',
              label: cost.estimated
                ? `${t('估算')} ${cost.baseline_monitor_count || 0}/${cost.monitor_count || 0}`
                : t('已校准'),
            },
          ],
        },
        {
          title: t('统计源'),
          value: unitMonitors.total,
          meta: `${t('启用')} ${unitMonitors.enabled} / ${unitMonitors.total}`,
          accent: 'cyan',
          progress: unitMonitors.total > 0 ? (unitMonitors.ok * 100) / unitMonitors.total : 0,
          tags: [
            { color: 'green', label: `${t('正常')} ${unitMonitors.ok}` },
            { color: 'red', label: `${t('异常')} ${unitMonitors.error}` },
            { color: 'grey', label: `${t('待检测')} ${unitMonitors.pending}` },
          ],
        },
        {
          title: t('余额汇总'),
          value: formatMoney(unitMonitors.current_balance),
          meta: `${t('已用')} ${formatMoney(unitMonitors.used_amount)}`,
          accent: 'purple',
          tags: [{ color: 'blue', label: t('平台账号余额') }],
        },
        {
          title: t('用户统计'),
          value: users.total,
          meta: `${rangeLabel}${t('活跃')} ${users.active_users || users.active_24h} · ${t('启用')} ${users.enabled}`,
          accent: 'purple',
          progress: users.total > 0 ? ((users.active_users || users.active_24h) * 100) / users.total : 0,
          tags: [
            { color: 'purple', label: `${t('管理员')} ${users.admin}` },
            { color: 'grey', label: `${t('停用')} ${users.disabled}` },
          ],
        },
        {
          title: t('渠道健康'),
          value: `${channels.enabled}/${channels.total}`,
          meta: `${t('已用额度')} ${channels.used_quota}`,
          accent: channels.unhealthy > 0 ? 'orange' : 'green',
          progress: channels.total > 0 ? (channels.healthy * 100) / channels.total : 0,
          tags: [
            { color: 'green', label: `${t('健康')} ${channels.healthy}` },
            { color: 'orange', label: `${t('降级')} ${channels.degraded}` },
            { color: 'red', label: `${t('异常')} ${channels.unhealthy}` },
          ],
        },
        {
          title: t('采集能力'),
          value: `Token ${unitMonitors.token_count}`,
          meta: `Model ${unitMonitors.model_count} · Group ${unitMonitors.group_count}`,
          accent: 'cyan',
          tags: [{ color: 'cyan', label: t('来自单位账号详情') }],
        },
        {
          title: `SFT ${rangeLabel}`,
          value: formatPercentPerm(sft.success_rate_perm),
          meta: `${t('成功')} ${sft.success_summary} · ${t('失败')} ${sft.failed_summary}`,
          accent: sft.failed_summary > 0 ? 'orange' : 'green',
          progress: Number(sft.success_rate_perm || 0) / 10,
          tags: [
            { color: 'blue', label: `51 ${sft.intercepted}` },
            { color: 'red', label: `52 ${sft.client_visible}` },
            { color: 'grey', label: `5 ${sft.legacy_error}` },
          ],
        },
        {
          title: t('模型活动'),
          value: (overview.models || []).length,
          meta:
            (overview.models || [])[0]?.model_name ||
            `${rangeLabel}${t('暂无模型调用')}`,
          accent: 'purple',
          tags: [{ color: 'violet', label: t('按请求数排行') }],
        },
        {
          title: t('最近刷新'),
          value: unitMonitors.last_checked_time
            ? timestamp2string(unitMonitors.last_checked_time)
            : '-',
          meta: overview.generated_at
            ? `${t('看板生成')} ${timestamp2string(overview.generated_at)}`
            : t('由单位账号监控检测时间生成'),
          accent: 'cyan',
          tags: [{ color: 'cyan', label: t('动态快照') }],
        },
      ];
      if (activeDashboard === 'model_performance') {
        return [
          commonCards[1],
          commonCards[10],
          {
            title: t('平均耗时'),
            value: formatSeconds(traffic.average_use_time),
            meta: `${t('流式')} ${formatPercentPerm(traffic.stream_rate_perm)} · ${t('成功率')} ${formatPercentPerm(traffic.success_rate_perm)}`,
            tags: [
              { color: 'green', label: `${t('成功')} ${traffic.success_count}` },
              { color: 'red', label: `${t('错误')} ${traffic.error_count}` },
            ],
          },
          {
            title: t('模型消耗'),
            value: formatQuota(traffic.tokens),
            meta: `${t('额度')} ${formatQuota(traffic.quota)}`,
            tags: [{ color: 'violet', label: t('模型维度') }],
          },
        ];
      }
      if (activeDashboard === 'user_consumption') {
        return [
          commonCards[2],
          commonCards[6],
          {
            title: t('用户额度'),
            value: formatQuota(users.quota),
            meta: `${t('已用')} ${formatQuota(users.used_quota)} · ${t('累计请求')} ${formatQuota(users.request_count)}`,
            tags: [{ color: 'purple', label: `${rangeLabel}${t('活跃')} ${users.active_users || users.active_24h}` }],
          },
          {
            title: t('错误影响'),
            value: traffic.user_visible_errors,
            meta: `${t('拦截')} ${traffic.intercepted_errors} · ${t('探测')} ${traffic.probe_count}`,
            tags: [{ color: 'red', label: t('用户侧体验') }],
          },
        ];
      }
      if (activeDashboard === 'cost_flow') {
        return [
          commonCards[3],
          {
            title: t('平台今日收入'),
            value: formatMoney(cost.platform_revenue_usd),
            meta: `${t('成功消费额度')} ${formatQuota(cost.platform_revenue_quota)}`,
            tags: [
              { color: 'green', label: '2 / 20' },
              { color: 'blue', label: t('用户侧流水') },
            ],
          },
          {
            title: t('上游今日成本'),
            value: formatMoney(cost.upstream_cost_usd),
            meta: `${t('快照覆盖')} ${cost.baseline_monitor_count || 0}/${cost.monitor_count || 0}`,
            tags: [
              { color: cost.estimated ? 'orange' : 'green', label: cost.estimated ? t('估算') : t('已校准') },
              { color: 'cyan', label: `${t('快照')} ${cost.snapshot_count || 0}` },
            ],
          },
          {
            title: t('预计今日利润'),
            value: formatMoney(cost.estimated_profit_usd),
            meta: `${t('毛利率')} ${formatPercentPerm(cost.gross_margin_perm)}`,
            tags: [
              { color: cost.estimated_profit_usd >= 0 ? 'green' : 'red', label: cost.estimated_profit_usd >= 0 ? t('盈利') : t('亏损') },
              { color: cost.missing_baseline_count > 0 ? 'orange' : 'blue', label: `${t('缺少基线')} ${cost.missing_baseline_count || 0}` },
            ],
          },
        ];
      }
      if (activeDashboard === 'unit_channel') {
        return [
          commonCards[4],
          commonCards[5],
          commonCards[7],
          commonCards[8],
        ];
      }
      return commonCards.filter((_, index) => index !== 3);
    },
    [activeDashboard, overview, rangeLabel, t],
  );

  const dashboardPanels = useMemo(() => {
    if (activeDashboard === 'model_performance') {
      return [
        {
          title: t('模型统计'),
          color: 'green',
          lines:
            (overview.models || []).length > 0
              ? (overview.models || []).slice(0, 4).map(
                  (model) =>
                    `${model.model_name || '-'} · ${t('请求')} ${model.request_count || 0} · ${t('Token')} ${formatQuota(model.tokens)} · ${t('额度')} ${formatQuota(model.quota)}`,
                )
              : [`${rangeLabel}${t('暂无模型调用')}`],
        },
        {
          title: t('性能摘要'),
          color: 'blue',
          lines: [
            `${t('平均耗时')} ${formatSeconds((overview.traffic || {}).average_use_time)} · ${t('流式')} ${formatPercentPerm((overview.traffic || {}).stream_rate_perm)}`,
            `${t('成功')} ${(overview.traffic || {}).success_count || 0} · ${t('错误')} ${(overview.traffic || {}).error_count || 0}`,
          ],
        },
      ];
    }
    if (activeDashboard === 'user_consumption') {
      return [
        {
          title: t('用户排行'),
          color: 'green',
          lines:
            (overview.top_users || []).length > 0
              ? (overview.top_users || []).slice(0, 4).map(
                  (user) =>
                    `${user.username || `#${user.user_id}`} · ${t('请求')} ${user.request_count || 0} · ${t('Token')} ${formatQuota(user.tokens)} · ${t('额度')} ${formatQuota(user.quota)}`,
                )
              : [`${rangeLabel}${t('暂无用户调用')}`],
        },
        {
          title: t('消耗结构'),
          color: 'purple',
          lines: [
            `${t('Token')} ${formatQuota((overview.traffic || {}).tokens)} · ${t('额度')} ${formatQuota((overview.traffic || {}).quota)}`,
            `${t('活跃用户')} ${(overview.users || {}).active_users || (overview.users || {}).active_24h || 0} · ${t('总用户')} ${(overview.users || {}).total || 0}`,
          ],
        },
      ];
    }
    if (activeDashboard === 'unit_channel') {
      const topChannels = ((rankData.channels || {}).items || overview.top_channels || []).slice(0, 4);
      const dimensionLabel =
        STATS_DIMENSION_OPTIONS.find((item) => item.value === statsDimension)?.label ||
        '分组';
      return [
        {
          title: t('渠道排行'),
          color: 'green',
          lines:
            topChannels.length > 0
              ? topChannels.map(
                  (channel) =>
                    `#${channel.channel_id} ${channel.channel_name || '-'} · ${channel.unit_name || t('未绑定单位')} · ${t('请求')} ${channel.request_count || 0} · ${t('额度')} ${formatQuota(channel.quota)}`,
                )
              : [`${rangeLabel}${t('暂无渠道调用')}`],
        },
        {
          title: t('单位采集'),
          color: 'cyan',
          lines: [
            `${t('正常')} ${(overview.unit_monitors || {}).ok || 0} · ${t('异常')} ${(overview.unit_monitors || {}).error || 0} · ${t('待检测')} ${(overview.unit_monitors || {}).pending || 0}`,
            `Token ${(overview.unit_monitors || {}).token_count || 0} · Model ${(overview.unit_monitors || {}).model_count || 0} · Group ${(overview.unit_monitors || {}).group_count || 0}`,
          ],
        },
        {
          title: t('渠道配置'),
          color: 'blue',
          lines: [
            `${t('启用')} ${(overview.channels || {}).enabled || 0}/${(overview.channels || {}).total || 0} · ${t('手动停用')} ${(overview.channels || {}).manually_disabled || 0} · ${t('自动停用')} ${(overview.channels || {}).auto_disabled || 0}`,
            `${t('健康')} ${(overview.channels || {}).healthy || 0} · ${t('降级')} ${(overview.channels || {}).degraded || 0} · ${t('异常')} ${(overview.channels || {}).unhealthy || 0}`,
          ],
        },
        {
          title: `${t('钻取维度')} · ${t(dimensionLabel)}`,
          color: 'purple',
          lines:
            (dimensionRankData.items || []).length > 0
              ? (dimensionRankData.items || []).slice(0, 3).map(
                  (item) =>
                    `${item.label || item.dimension_key || `#${item.dimension_id}`} · ${t('请求')} ${formatQuota(item.request_count)} · ${t('成功率')} ${formatPercentPerm(item.success_rate_perm)}`,
                )
              : [`${rangeLabel}${t('暂无维度排行数据')}`],
        },
      ];
    }
    if (activeDashboard === 'cost_flow') {
      const topCostItems = (costRankData.items || []).slice(0, 3);
      const costDimensionLabel =
        COST_DIMENSION_OPTIONS.find((item) => item.value === costDimension)?.label ||
        '单位/账号';
      return [
        {
          title: t('收入口径'),
          color: 'green',
          status: t('实时'),
          lines: [
            `${t('平台收入')} = ${t('今日成功消费日志')} 2/20 ${t('的额度')} / QuotaPerUnit`,
            `${t('当前收入')} ${formatMoney((overview.cost || {}).platform_revenue_usd)} · ${t('额度')} ${formatQuota((overview.cost || {}).platform_revenue_quota)}`,
            `${t('统计范围')} ${timestamp2string((overview.cost || {}).day_start || 0)} - ${timestamp2string((overview.cost || {}).day_end || 0)}`,
          ],
        },
        {
          title: t('成本口径'),
          color: (overview.cost || {}).estimated ? 'orange' : 'green',
          status: (overview.cost || {}).estimated ? t('估算') : t('已校准'),
          lines: [
            `${t('上游成本')} = ${t('今日最新快照已用额度')} - ${t('今日首个快照已用额度')}`,
            `${t('当前成本')} ${formatMoney((overview.cost || {}).upstream_cost_usd)} · ${t('快照覆盖')} ${(overview.cost || {}).baseline_monitor_count || 0}/${(overview.cost || {}).monitor_count || 0}`,
            (overview.cost || {}).note || t('暂无成本口径说明'),
          ],
        },
        {
          title: t('利润口径'),
          color: (overview.cost || {}).estimated_profit_usd >= 0 ? 'green' : 'red',
          status: t('今日'),
          lines: [
            `${t('预计利润')} = ${t('平台收入')} - ${t('上游成本')}`,
            `${t('预计利润')} ${formatMoney((overview.cost || {}).estimated_profit_usd)} · ${t('毛利率')} ${formatPercentPerm((overview.cost || {}).gross_margin_perm)}`,
            `${t('缺少基线')} ${(overview.cost || {}).missing_baseline_count || 0} · ${t('快照数')} ${(overview.cost || {}).snapshot_count || 0}`,
          ],
        },
        {
          title: `${t('成本排行')} · ${t(costDimensionLabel)}`,
          color: 'orange',
          status: t('今日'),
          lines:
            topCostItems.length > 0
              ? topCostItems.map(
                  (item) =>
                    `${item.label || item.key || '-'} · ${t('收入')} ${formatMoney(item.platform_revenue_usd)} · ${t('成本')} ${formatMoney(item.upstream_cost_usd)} · ${t('利润')} ${formatMoney(item.estimated_profit_usd)}`,
                )
              : [t('暂无流水成本明细')],
        },
      ];
    }
    return [
      {
        title: t('用户统计'),
        color: 'green',
        lines:
          (overview.top_users || []).length > 0
            ? [
                `${t('总用户')} ${(overview.users || {}).total || 0} · ${rangeLabel}${t('活跃')} ${(overview.users || {}).active_users || (overview.users || {}).active_24h || 0}`,
                ...(overview.top_users || []).slice(0, 3).map(
                  (user) =>
                    `${user.username || `#${user.user_id}`} · ${t('请求')} ${user.request_count || 0} · ${t('Token')} ${formatQuota(user.tokens)} · ${t('额度')} ${formatQuota(user.quota)}`,
                ),
              ]
            : [
                `${t('总用户')} ${(overview.users || {}).total || 0} · ${t('启用')} ${(overview.users || {}).enabled || 0}`,
                `${rangeLabel}${t('暂无用户调用')}`,
              ],
      },
      {
        title: t('模型统计'),
        color: 'green',
        lines:
          (overview.models || []).length > 0
            ? (overview.models || []).slice(0, 3).map(
                (model) =>
                  `${model.model_name || '-'} · ${t('请求')} ${model.request_count || 0} · ${t('Token')} ${formatQuota(model.tokens)} · ${t('额度')} ${formatQuota(model.quota)}`,
              )
            : [`${rangeLabel}${t('暂无模型调用')}`],
      },
      {
        title: t('渠道健康'),
        color: 'green',
        lines:
          (overview.top_channels || []).length > 0
            ? [
                `${t('启用')} ${(overview.channels || {}).enabled || 0} / ${(overview.channels || {}).total || 0}`,
                ...(overview.top_channels || []).slice(0, 3).map(
                  (channel) =>
                    `#${channel.channel_id} ${channel.channel_name || '-'} · ${t('请求')} ${channel.request_count || 0} · ${t('Token')} ${formatQuota(channel.tokens)} · ${t('额度')} ${formatQuota(channel.quota)}`,
                ),
              ]
            : [
                `${t('启用')} ${(overview.channels || {}).enabled || 0} / ${(overview.channels || {}).total || 0}`,
                `${t('手动停用')} ${(overview.channels || {}).manually_disabled || 0} · ${t('自动停用')} ${(overview.channels || {}).auto_disabled || 0}`,
              ],
      },
      {
        title: t('容错重试'),
        color: 'green',
        lines: [
          `20 ${((overview.sft || {}).success_summary || 0)} · 50 ${((overview.sft || {}).failed_summary || 0)} · ${t('成功率')} ${formatPercentPerm((overview.sft || {}).success_rate_perm)}`,
          `29 ${((overview.sft || {}).probe_success || 0)} · 59 ${((overview.sft || {}).probe_failed || 0)} · 51 ${((overview.sft || {}).intercepted || 0)} · 52 ${((overview.sft || {}).client_visible || 0)}`,
        ],
      },
    ];
  }, [activeDashboard, costDimension, costRankData.items, dimensionRankData.items, overview, rangeLabel, rankData.channels, statsDimension, t]);

  const renderRateTags = (record) => (
    <Space spacing={4} wrap>
      <Tag color='green' size='small'>
        {formatPercentPerm(record.success_rate_perm)}
      </Tag>
      <Tag color='blue' size='small'>
        {t('流式')} {formatPercentPerm(record.stream_rate_perm)}
      </Tag>
    </Space>
  );

  const renderSftSplit = (record) => (
    <Space spacing={4} wrap>
      <Tag color='green' size='small'>
        2 {record.direct_success_count || 0}
      </Tag>
      <Tag color='cyan' size='small'>
        20 {record.retry_success_count || 0}
      </Tag>
      <Tag color='red' size='small'>
        50 {record.retry_failed_count || 0}
      </Tag>
      <Tag color='grey' size='small'>
        5 {record.legacy_error_count || 0}
      </Tag>
    </Space>
  );

  const commonRankColumns = [
    {
      title: t('请求'),
      dataIndex: 'request_count',
      width: 110,
      render: (value) => formatQuota(value),
    },
    {
      title: t('成功/失败'),
      dataIndex: 'success_count',
      width: 140,
      render: (_, record) => (
        <Space spacing={4} wrap>
          <Tag color='green' size='small'>
            {record.success_count || 0}
          </Tag>
          <Tag color='red' size='small'>
            {record.error_count || 0}
          </Tag>
        </Space>
      ),
    },
    {
      title: t('类型拆分'),
      dataIndex: 'direct_success_count',
      width: 220,
      render: (_, record) => renderSftSplit(record),
    },
    {
      title: t('成功率/流式'),
      dataIndex: 'success_rate_perm',
      width: 170,
      render: (_, record) => renderRateTags(record),
    },
    {
      title: t('平均耗时'),
      dataIndex: 'average_use_time',
      width: 110,
      render: (value) => formatSeconds(value),
    },
    {
      title: 'Token',
      dataIndex: 'tokens',
      width: 130,
      render: (value) => formatQuota(value),
    },
    {
      title: t('额度'),
      dataIndex: 'quota',
      width: 130,
      render: (value) => formatQuota(value),
    },
  ];

  const getRankColumns = () => {
    if (activeDashboard === 'model_performance') {
      return [
        {
          title: t('模型'),
          dataIndex: 'model_name',
          width: 280,
          fixed: 'left',
          render: (value) => (
            <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 260 }}>
              {value || '-'}
            </Text>
          ),
        },
        ...commonRankColumns,
      ];
    }
    if (activeDashboard === 'user_consumption') {
      return [
        {
          title: t('用户'),
          dataIndex: 'username',
          width: 240,
          fixed: 'left',
          render: (value, record) => (
            <div>
              <Text>{value || `#${record.user_id}`}</Text>
              <div className='text-xs text-gray-500'>ID: {record.user_id || '-'}</div>
            </div>
          ),
        },
        ...commonRankColumns,
      ];
    }
    return [
      {
        title: t('渠道'),
        dataIndex: 'channel_name',
        width: 300,
        fixed: 'left',
        render: (value, record) => (
          <div>
            <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 260 }}>
              #{record.channel_id} {value || '-'}
            </Text>
            <div className='text-xs text-gray-500'>
              {t('优先级')} {record.priority || 0} · {t('权重')} {record.weight || 0}
            </div>
          </div>
        ),
      },
      {
        title: t('所属单位/账号'),
        dataIndex: 'unit_name',
        width: 260,
        render: (_, record) => (
          <div>
            <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 240 }}>
              {record.unit_name || t('未绑定单位')}
            </Text>
            <div className='text-xs text-gray-500'>
              {record.unit_type || '-'} · {record.unit_account_name || t('未绑定账号')}
            </div>
          </div>
        ),
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        width: 150,
        render: (_, record) => (
          <Space spacing={4} wrap>
            <Tag color={record.status === 1 ? 'green' : 'grey'} size='small'>
              {record.status === 1 ? t('启用') : t('停用')}
            </Tag>
            <Tag
              color={
                record.health_status === 'unhealthy'
                  ? 'red'
                  : record.health_status === 'degraded'
                    ? 'orange'
                    : 'green'
              }
              size='small'
            >
              {record.health_status || 'healthy'}
            </Tag>
          </Space>
        ),
      },
      ...commonRankColumns,
    ];
  };

  const renderRankCard = () => {
    if (!activeRankKey) return null;
    const data = rankData[activeRankKey] || emptyRankData;
    const titleMap = {
      models: t('模型完整排行'),
      users: t('用户消耗排行'),
      channels: t('渠道调用排行'),
    };
    return (
      <CardPro
        type='type3'
        actionsArea={
          <Space wrap>
            <Text strong>{titleMap[activeRankKey]}</Text>
            <Select
              prefix={t('排序')}
              value={rankSort}
              onChange={(value) => setRankSort(value || 'requests')}
              optionList={[
                { label: t('请求数'), value: 'requests' },
                { label: 'Token', value: 'tokens' },
                { label: t('额度'), value: 'quota' },
                { label: t('错误数'), value: 'errors' },
                { label: t('平均耗时'), value: 'latency' },
                { label: t('成功数'), value: 'success' },
              ]}
              style={{ width: 150 }}
            />
          </Space>
        }
        paginationArea={createCardProPagination({
          currentPage: data.page || rankPage,
          pageSize: data.page_size || rankPageSize,
          total: data.total || 0,
          onPageChange: (page) => queryRank(activeRankKey, page, data.page_size || rankPageSize),
          onPageSizeChange: (size) => queryRank(activeRankKey, 1, size),
          isMobile,
          t,
        })}
        t={t}
      >
        <Table
          rowKey={(record) =>
            record.model_name || record.user_id || record.channel_id
          }
          columns={getRankColumns()}
          dataSource={data.items || []}
          loading={rankLoading}
          pagination={false}
          scroll={{ x: activeDashboard === 'unit_channel' ? 1780 : 1360 }}
          empty={t('暂无排行数据')}
        />
      </CardPro>
    );
  };

  const getDimensionRankColumns = () => [
    {
      title: t('维度对象'),
      dataIndex: 'label',
      width: 300,
      fixed: 'left',
      render: (value, record) => (
        <div>
          <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 280 }}>
            {value ||
              record.dimension_key ||
              (record.dimension_id ? `#${record.dimension_id}` : '-')}
          </Text>
          <div className='text-xs text-gray-500'>
            {statsDimension === 'token' && `Token ID: ${record.dimension_id || '-'}`}
            {statsDimension !== 'token' && `${t('键')}：${record.dimension_key || '-'}`}
          </div>
        </div>
      ),
    },
    ...commonRankColumns,
  ];

  const renderDimensionRankCard = () => {
    if (activeDashboard === 'cost_flow') return null;
    const dimensionLabel =
      STATS_DIMENSION_OPTIONS.find((item) => item.value === statsDimension)?.label ||
      '分组';
    return (
      <CardPro
        type='type3'
        actionsArea={
          <Space wrap>
            <Text strong>{t('多维度钻取排行')}</Text>
            <Select
              prefix={t('维度')}
              value={statsDimension}
              onChange={(value) => setStatsDimension(value || 'group')}
              optionList={STATS_DIMENSION_OPTIONS.map((item) => ({
                label: t(item.label),
                value: item.value,
              }))}
              style={{ width: 150 }}
            />
            <Select
              prefix={t('排序')}
              value={dimensionSort}
              onChange={(value) => setDimensionSort(value || 'requests')}
              optionList={[
                { label: t('请求数'), value: 'requests' },
                { label: 'Token', value: 'tokens' },
                { label: t('额度'), value: 'quota' },
                { label: t('错误数'), value: 'errors' },
                { label: t('平均耗时'), value: 'latency' },
                { label: t('成功数'), value: 'success' },
              ]}
              style={{ width: 150 }}
            />
            <Tag color='cyan' size='small'>
              {t(dimensionLabel)}
            </Tag>
          </Space>
        }
        paginationArea={createCardProPagination({
          currentPage: dimensionRankData.page || dimensionRankPage,
          pageSize: dimensionRankData.page_size || dimensionRankPageSize,
          total: dimensionRankData.total || 0,
          onPageChange: (page) =>
            queryDimensionRank(page, dimensionRankData.page_size || dimensionRankPageSize),
          onPageSizeChange: (size) => queryDimensionRank(1, size),
          isMobile,
          t,
        })}
        t={t}
      >
        <Table
          rowKey={(record) =>
            `${statsDimension}-${record.dimension_id || record.dimension_key || record.label}`
          }
          columns={getDimensionRankColumns()}
          dataSource={dimensionRankData.items || []}
          loading={dimensionRankLoading}
          pagination={false}
          scroll={{ x: 1380 }}
          empty={t('暂无维度排行数据')}
        />
      </CardPro>
    );
  };

  const renderModelCoverageCard = () => {
    if (activeDashboard !== 'model_performance') return null;
    const riskColor = {
      ok: 'green',
      idle: 'grey',
      warning: 'orange',
      critical: 'red',
    };
    const columns = [
      {
        title: t('模型'),
        dataIndex: 'model_name',
        width: 280,
        fixed: 'left',
        render: (value, record) => (
          <div>
            <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 260 }}>
              {value || '-'}
            </Text>
            <div className='text-xs text-gray-500'>
              {(record.groups || []).slice(0, 4).join(' / ') || t('暂无分组')}
              {(record.groups || []).length > 4 ? ' ...' : ''}
            </div>
          </div>
        ),
      },
      {
        title: t('覆盖状态'),
        dataIndex: 'risk_level',
        width: 190,
        render: (value, record) => (
          <div>
            <Tag color={riskColor[value] || 'grey'} size='small'>
              {value === 'ok'
                ? t('正常')
                : value === 'idle'
                  ? t('空闲')
                  : value === 'critical'
                    ? t('高危')
                    : t('关注')}
            </Tag>
            <div className='text-xs text-gray-500' style={{ marginTop: 4 }}>
              <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 170 }}>
                {record.risk_reason || '-'}
              </Text>
            </div>
          </div>
        ),
      },
      {
        title: t('分组覆盖'),
        dataIndex: 'group_count',
        width: 150,
        render: (_, record) => (
          <Space spacing={4} wrap>
            <Tag color='blue' size='small'>
              {t('启用')} {record.enabled_group_count || 0}
            </Tag>
            <Tag color='grey' size='small'>
              {t('总计')} {record.group_count || 0}
            </Tag>
          </Space>
        ),
      },
      {
        title: t('渠道覆盖'),
        dataIndex: 'enabled_channel_count',
        width: 230,
        render: (_, record) => (
          <div>
            <Space spacing={4} wrap>
              <Tag color='green' size='small'>
                {t('健康')} {record.healthy_channel_count || 0}
              </Tag>
              <Tag color='orange' size='small'>
                {t('降级')} {record.degraded_channel_count || 0}
              </Tag>
              <Tag color='red' size='small'>
                {t('停用')} {record.disabled_channel_count || 0}
              </Tag>
            </Space>
            <div className='text-xs text-gray-500' style={{ marginTop: 4 }}>
              {t('启用覆盖')} {formatPercentPerm(record.coverage_rate_perm)} · {t('健康率')}{' '}
              {formatPercentPerm(record.healthy_rate_perm)}
            </div>
          </div>
        ),
      },
      {
        title: t('最近请求'),
        dataIndex: 'request_count',
        width: 180,
        render: (_, record) => (
          <Space spacing={4} wrap>
            <Tag color='blue' size='small'>
              {formatQuota(record.request_count)}
            </Tag>
            <Tag color={Number(record.error_count || 0) > 0 ? 'red' : 'green'} size='small'>
              {t('错误')} {formatQuota(record.error_count)}
            </Tag>
          </Space>
        ),
      },
      {
        title: t('成功率/耗时'),
        dataIndex: 'success_rate_perm',
        width: 170,
        render: (_, record) => (
          <div>
            <Text>{formatPercentPerm(record.success_rate_perm)}</Text>
            <div className='text-xs text-gray-500'>
              {t('平均耗时')} {record.average_use_time || 0}s
            </div>
          </div>
        ),
      },
      {
        title: 'Token / 额度',
        dataIndex: 'tokens',
        width: 180,
        render: (_, record) => (
          <div>
            <Text>Token {formatQuota(record.tokens)}</Text>
            <div className='text-xs text-gray-500'>{t('额度')} {formatQuota(record.quota)}</div>
          </div>
        ),
      },
      {
        title: t('样本渠道'),
        dataIndex: 'sample_channels',
        width: 320,
        render: (value = []) => (
          <Space wrap spacing={4}>
            {value.slice(0, 5).map((channel) => (
              <Tag
                key={`${channel.channel_id}-${channel.group}`}
                color={channel.health_status === 'healthy' ? 'green' : 'grey'}
                size='small'
              >
                #{channel.channel_id} {channel.channel_name || '-'}
              </Tag>
            ))}
            {value.length > 5 && (
              <Tag color='grey' size='small'>
                +{value.length - 5}
              </Tag>
            )}
          </Space>
        ),
      },
    ];

    return (
      <CardPro
        type='type3'
        actionsArea={
          <Space wrap>
            <Text strong>{t('模型覆盖与缺口矩阵')}</Text>
            <Select
              prefix={t('排序')}
              value={coverageSort}
              onChange={(value) => setCoverageSort(value || 'coverage')}
              optionList={[
                { label: t('覆盖风险'), value: 'coverage' },
                { label: t('健康率'), value: 'healthy' },
                { label: t('启用渠道少'), value: 'channels' },
                { label: t('请求数'), value: 'requests' },
                { label: t('错误数'), value: 'errors' },
                { label: t('平均耗时'), value: 'latency' },
                { label: 'Token', value: 'tokens' },
                { label: t('额度'), value: 'quota' },
              ]}
              style={{ width: 170 }}
            />
            <Tag color='purple' size='small'>
              {t('能力覆盖')}
            </Tag>
          </Space>
        }
        paginationArea={createCardProPagination({
          currentPage: coverageData.page || coveragePage,
          pageSize: coverageData.page_size || coveragePageSize,
          total: coverageData.total || 0,
          onPageChange: (page) =>
            queryModelCoverage(page, coverageData.page_size || coveragePageSize),
          onPageSizeChange: (size) => queryModelCoverage(1, size),
          isMobile,
          t,
        })}
        t={t}
      >
        <Table
          rowKey={(record) => record.model_name}
          columns={columns}
          dataSource={coverageData.items || []}
          loading={coverageLoading}
          pagination={false}
          scroll={{ x: 1700 }}
          empty={t('暂无模型覆盖数据')}
        />
      </CardPro>
    );
  };

  const renderCostSource = (record) => {
    if (record.cost_source === 'snapshot_delta') {
      return (
        <Tag color={record.baseline_ready ? 'green' : 'orange'} size='small'>
          {record.baseline_ready ? t('快照差值') : t('快照不足')}
        </Tag>
      );
    }
    if (record.cost_source === 'missing_monitor') {
      return (
        <Tag color='red' size='small'>
          {t('缺少统计源')}
        </Tag>
      );
    }
    return (
      <Tag color='orange' size='small'>
        {t('额度分摊')}
      </Tag>
    );
  };

  const getCostRankColumns = () => [
    {
      title: t('对象'),
      dataIndex: 'label',
      width: 300,
      fixed: 'left',
      render: (value, record) => (
        <div>
          <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 280 }}>
            {value || record.key || '-'}
          </Text>
          <div className='text-xs text-gray-500'>
            {record.dimension === 'unit' &&
              `${record.unit_type || '-'} · ${t('单位')} #${record.unit_id || '-'} · ${t('账号')} #${record.unit_account_id || '-'}`}
            {record.dimension === 'channel' &&
              `${t('渠道')} #${record.channel_id || '-'} · ${record.channel_health_status || 'healthy'}`}
            {record.dimension === 'model' && `${t('模型')} ${record.model_name || '-'}`}
            {record.dimension === 'user' && `ID: ${record.user_id || '-'}`}
          </div>
        </div>
      ),
    },
    {
      title: t('归属'),
      dataIndex: 'unit_name',
      width: 260,
      render: (_, record) => (
        <div>
          <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 240 }}>
            {record.unit_name || (record.dimension === 'unit' ? record.label : '-')}
          </Text>
          <div className='text-xs text-gray-500'>
            {record.account || '-'}
            {record.dimension === 'channel' && (
              <>
                {' · '}
                {t('优先级')} {record.priority || 0} · {t('权重')} {record.weight || 0}
              </>
            )}
          </div>
        </div>
      ),
    },
    {
      title: t('请求/Token'),
      dataIndex: 'request_count',
      width: 150,
      render: (_, record) => (
        <Space spacing={4} wrap>
          <Tag color='blue' size='small'>
            {t('请求')} {formatQuota(record.request_count)}
          </Tag>
          <Tag color='green' size='small'>
            Token {formatQuota(record.tokens)}
          </Tag>
        </Space>
      ),
    },
    {
      title: t('收入'),
      dataIndex: 'platform_revenue_usd',
      width: 150,
      render: (value, record) => (
        <div>
          <Text>{formatMoney(value)}</Text>
          <div className='text-xs text-gray-500'>{t('额度')} {formatQuota(record.quota)}</div>
        </div>
      ),
    },
    {
      title: t('上游成本'),
      dataIndex: 'upstream_cost_usd',
      width: 150,
      render: (value) => formatMoney(value),
    },
    {
      title: t('预计利润'),
      dataIndex: 'estimated_profit_usd',
      width: 150,
      render: (value, record) => (
        <Space spacing={4} wrap>
          <Tag color={Number(value || 0) >= 0 ? 'green' : 'red'} size='small'>
            {formatMoney(value)}
          </Tag>
          <Tag color='purple' size='small'>
            {formatPercentPerm(record.gross_margin_perm)}
          </Tag>
        </Space>
      ),
    },
    {
      title: t('成本口径'),
      dataIndex: 'cost_source',
      width: 170,
      render: (_, record) => (
        <div>
          {renderCostSource(record)}
          <div className='text-xs text-gray-500' style={{ marginTop: 4 }}>
            {record.estimated ? t('估算') : t('精确')} · {record.platform_status || '-'}
          </div>
        </div>
      ),
    },
    {
      title: t('余额/已用'),
      dataIndex: 'current_balance',
      width: 170,
      render: (_, record) => (
        <div>
          <Text>{formatMoney(record.current_balance, record.balance_unit)}</Text>
          <div className='text-xs text-gray-500'>
            {t('已用')} {formatMoney(record.used_amount, record.balance_unit)}
          </div>
        </div>
      ),
    },
    {
      title: t('快照'),
      dataIndex: 'snapshot_count',
      width: 180,
      render: (_, record) => (
        <div>
          <Text>{formatQuota(record.snapshot_count || 0)}</Text>
          <div className='text-xs text-gray-500'>
            {record.last_snapshot_time
              ? timestamp2string(record.last_snapshot_time)
              : record.last_checked_time
                ? timestamp2string(record.last_checked_time)
                : '-'}
          </div>
        </div>
      ),
    },
    {
      title: t('说明'),
      dataIndex: 'missing_snapshot_reason',
      width: 280,
      render: (value) => (
        <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 260 }}>
          {value || '-'}
        </Text>
      ),
    },
  ];

  const renderCostRankCard = () => {
    if (activeDashboard !== 'cost_flow') return null;
    const dimensionLabel =
      COST_DIMENSION_OPTIONS.find((item) => item.value === costDimension)?.label ||
      '单位/账号';
    return (
      <CardPro
        type='type3'
        actionsArea={
          <Space wrap>
            <Text strong>{t('流水成本明细')}</Text>
            <Select
              prefix={t('维度')}
              value={costDimension}
              onChange={(value) => setCostDimension(value || 'unit')}
              optionList={COST_DIMENSION_OPTIONS.map((item) => ({
                label: t(item.label),
                value: item.value,
              }))}
              style={{ width: 160 }}
            />
            <Select
              prefix={t('排序')}
              value={costSort}
              onChange={(value) => setCostSort(value || 'revenue')}
              optionList={[
                { label: t('收入'), value: 'revenue' },
                { label: t('成本'), value: 'cost' },
                { label: t('利润'), value: 'profit' },
                { label: t('毛利率'), value: 'margin' },
                { label: t('请求数'), value: 'requests' },
                { label: 'Token', value: 'tokens' },
                { label: t('额度'), value: 'quota' },
              ]}
              style={{ width: 150 }}
            />
            <Tag color='orange' size='small'>
              {t(dimensionLabel)}
            </Tag>
          </Space>
        }
        paginationArea={createCardProPagination({
          currentPage: costRankData.page || costRankPage,
          pageSize: costRankData.page_size || costRankPageSize,
          total: costRankData.total || 0,
          onPageChange: (page) =>
            queryCostRank(page, costRankData.page_size || costRankPageSize),
          onPageSizeChange: (size) => queryCostRank(1, size),
          isMobile,
          t,
        })}
        t={t}
      >
        <Table
          rowKey={(record) => `${record.dimension}-${record.key}`}
          columns={getCostRankColumns()}
          dataSource={costRankData.items || []}
          loading={costRankLoading}
          pagination={false}
          scroll={{ x: 1800 }}
          empty={t('暂无流水成本明细')}
        />
      </CardPro>
    );
  };

  const actionsArea = (
    <Space wrap>
      <Select
        prefix={t('范围')}
        value={rangeHours}
        onChange={(value) => setRangeHours(Number(value) || 24)}
        optionList={RANGE_OPTIONS.map((item) => ({
          label: t(item.label),
          value: item.value,
        }))}
        style={{ width: 150 }}
      />
      <Button icon={<IconPlus />} type='primary' onClick={openCreateModal}>
        {t('新增统计源')}
      </Button>
      <Button icon={<IconRefresh />} onClick={checkAll} loading={loading}>
        {t('刷新全部')}
      </Button>
      <Input
        prefix={<IconSearch />}
        placeholder={t('搜索统计源/账号')}
        value={keyword}
        onChange={(value) => setKeyword(value)}
        showClear
        style={{ width: 220 }}
      />
      <Select
        placeholder={t('状态')}
        value={statusFilter}
        onChange={(value) => setStatusFilter(value || '')}
        optionList={[{ label: t('全部'), value: '' }, ...STATUS_OPTIONS]}
        style={{ width: 120 }}
      />
    </Space>
  );

  const showMonitorTable =
    activeDashboard === 'overview' ||
    activeDashboard === 'unit_channel' ||
    activeDashboard === 'cost_flow';

  const columns = [
    {
      title: t('ID'),
      dataIndex: 'id',
      width: 80,
    },
    {
      title: t('统计源名称'),
      dataIndex: 'name',
      width: 220,
      render: (value, record) => (
        <div>
          <Text strong>{value || '-'}</Text>
          <div className='text-xs text-gray-500'>
            {record.unit_name || '-'} · {record.unit_type || '-'}
          </div>
        </div>
      ),
    },
    {
      title: t('账号'),
      dataIndex: 'account',
      width: 220,
      render: (value, record) => (
        <div>
          <Text>{value || '-'}</Text>
          <div className='text-xs text-gray-500'>ID: {record.account_id || '-'}</div>
        </div>
      ),
    },
    {
      title: t('余额'),
      dataIndex: 'current_balance',
      width: 150,
      render: (value, record) => formatMoney(value, record.balance_unit),
    },
    {
      title: t('已用'),
      dataIndex: 'used_amount',
      width: 150,
      render: (value, record) => formatMoney(value, record.balance_unit),
    },
    {
      title: t('上游用户'),
      dataIndex: 'upstream_username',
      width: 180,
      render: (value, record) => (
        <div>
          <Text>{value || '-'}</Text>
          <div className='text-xs text-gray-500'>{record.upstream_user_id || '-'}</div>
        </div>
      ),
    },
    {
      title: t('信息'),
      dataIndex: 'token_count',
      width: 180,
      render: (_, record) => (
        <Space>
          <Tag color='blue' size='small'>
            Token {record.token_count || 0}
          </Tag>
          <Tag color='green' size='small'>
            Model {record.model_count || 0}
          </Tag>
          <Tag color='purple' size='small'>
            Group {record.group_count || 0}
          </Tag>
        </Space>
      ),
    },
    {
      title: t('状态'),
      dataIndex: 'platform_status',
      width: 120,
      render: (value) => <MonitorStatusTag value={value} />,
    },
    {
      title: t('最后检查'),
      dataIndex: 'last_checked_time',
      width: 180,
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
    {
      title: t('错误'),
      dataIndex: 'error_message',
      width: 260,
      render: (value) => (
        <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 240 }}>
          {value || '-'}
        </Text>
      ),
    },
    {
      title: t('操作'),
      dataIndex: 'operate',
      fixed: 'right',
      width: 250,
      render: (_, record) => (
        <Space>
          <Button
            size='small'
            icon={<IconRefresh />}
            loading={!!checkingIds[record.id]}
            onClick={() => checkMonitor(record.id)}
          >
            {t('刷新')}
          </Button>
          <Button
            size='small'
            type={record.status === 1 ? 'warning' : 'primary'}
            onClick={() =>
              updateMonitorStatus(record, record.status === 1 ? 2 : 1)
            }
          >
            {record.status === 1 ? t('停用') : t('启用')}
          </Button>
          <Button size='small' onClick={() => setDetailRecord(record)}>
            {t('详情')}
          </Button>
          <Popconfirm
            title={t('确定删除该统计源？')}
            content={t('删除后不会删除原单位和账号。')}
            onConfirm={() => deleteMonitor(record)}
          >
            <Button size='small' type='danger'>
              {t('删除')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <>
      {renderDashboardHeader()}

      <Tabs
        activeKey={activeDashboard}
        type='button'
        onChange={setActiveDashboard}
        style={{ marginBottom: 12 }}
      >
        {DASHBOARD_TABS.map((item) => (
          <Tabs.TabPane key={item.value} itemKey={item.value} tab={t(item.label)} />
        ))}
      </Tabs>

      {renderCardsGrid(dashboardCards)}

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: isMobile
            ? '1fr'
            : 'repeat(3, minmax(220px, 1fr))',
          gap: 12,
          marginBottom: 14,
        }}
      >
        {activeDashboard === 'overview' && (
          <>
            {renderTrendPanel(t('请求趋势'), 'request_count', 'blue')}
            {renderTrendPanel(t('错误趋势'), 'error_count', 'red')}
            {renderTrendPanel(t('Token 趋势'), 'tokens', 'green')}
          </>
        )}
        {activeDashboard === 'model_performance' && (
          <>
            {renderTrendPanel(t('请求趋势'), 'request_count', 'blue')}
            {renderTrendPanel(t('成功趋势'), 'success_count', 'green')}
            {renderTrendPanel(t('错误趋势'), 'error_count', 'red')}
          </>
        )}
        {activeDashboard === 'user_consumption' && (
          <>
            {renderTrendPanel(t('Token 趋势'), 'tokens', 'green')}
            {renderTrendPanel(t('额度趋势'), 'quota', 'purple')}
            {renderTrendPanel(t('请求趋势'), 'request_count', 'blue')}
          </>
        )}
        {activeDashboard === 'cost_flow' && (
          <>
            {renderTrendPanel(t('收入额度趋势'), 'quota', 'purple')}
            {renderTrendPanel(t('成功请求趋势'), 'success_count', 'green')}
            {renderTrendPanel(t('错误影响趋势'), 'error_count', 'red')}
          </>
        )}
        {activeDashboard === 'unit_channel' && (
          <>
            {renderTrendPanel(t('渠道请求趋势'), 'request_count', 'blue')}
            {renderTrendPanel(t('探测趋势'), 'probe_count', 'orange')}
            {renderTrendPanel(t('错误趋势'), 'error_count', 'red')}
          </>
        )}
      </div>

      {renderPanelsGrid(dashboardPanels)}

      {renderRankCard()}

      {renderDimensionRankCard()}

      {renderModelCoverageCard()}

      {renderCostRankCard()}

      {showMonitorTable && (
        <CardPro
          type='type3'
          actionsArea={actionsArea}
          paginationArea={createCardProPagination({
            currentPage: activePage,
            pageSize,
            total,
            onPageChange: (page) => queryMonitors(page, pageSize),
            onPageSizeChange: (size) => queryMonitors(1, size),
            isMobile,
            t,
          })}
          t={t}
        >
          <Table
            rowKey='id'
            columns={columns}
            dataSource={monitors}
            loading={loading}
            pagination={false}
            scroll={{ x: 2060 }}
            empty={t('暂无NACP统计')}
          />
        </CardPro>
      )}

      <Modal
        title={t('新增单位账号统计源')}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => document.querySelector('#nacp-stats-create-submit')?.click()}
        confirmLoading={modalLoading}
        okText={t('创建并立即检测')}
        cancelText={t('取消')}
        width={640}
      >
        <Form onSubmit={submitMonitor}>
          <Form.Input field='name' label={t('统计源名称')} placeholder={t('留空则使用单位和账号名称')} showClear />
          <Form.Select
            field='unit_id'
            label={t('所属单位')}
            placeholder={t('请选择所属单位')}
            optionList={unitOptions}
            style={{ width: '100%' }}
            rules={[{ required: true, message: t('请选择所属单位') }]}
            onChange={(value) => {
              setSelectedUnitId(value);
              loadAccounts(value).then();
            }}
          />
          <Form.Select
            field='unit_account_id'
            label={t('所属账号')}
            placeholder={
              selectedUnitId ? t('请选择所属账号') : t('请先选择所属单位')
            }
            optionList={accountOptions}
            disabled={!selectedUnitId}
            style={{ width: '100%' }}
            rules={[{ required: true, message: t('请选择所属账号') }]}
          />
          <button id='nacp-stats-create-submit' type='submit' hidden />
        </Form>
      </Modal>

      <Modal
        title={t('NACP统计详情')}
        visible={!!detailRecord}
        onCancel={() => setDetailRecord(null)}
        footer={
          <Space>
            <Button onClick={() => setDetailRecord(null)}>{t('关闭')}</Button>
            <Button type='primary' onClick={copyDetail}>
              {t('复制原始信息')}
            </Button>
          </Space>
        }
        width={760}
      >
        {detailRecord && (
          <div>
            <div>
              <Text strong>{detailRecord.name || '-'}</Text>
              <div className='text-xs text-gray-500'>
                {detailRecord.unit_name || '-'} · {detailRecord.account || '-'}
              </div>
            </div>
            <Tabs defaultActiveKey='summary' size='medium' style={{ marginTop: 16 }}>
              <Tabs.TabPane tab={t('摘要')} itemKey='summary'>
                {renderDetailSummary(detailRecord)}
              </Tabs.TabPane>
              <Tabs.TabPane tab='JSON' itemKey='json'>
                <pre
                  style={{
                    maxHeight: 460,
                    overflow: 'auto',
                    padding: 12,
                    borderRadius: 8,
                    background: 'rgba(15, 23, 42, 0.92)',
                    color: '#e5e7eb',
                    fontSize: 12,
                    lineHeight: 1.6,
                  }}
                >
                  {formatRawJSON(detailRecord.raw_json)}
                </pre>
              </Tabs.TabPane>
            </Tabs>
          </div>
        )}
      </Modal>
    </>
  );
};

export default NacpStatsTable;
