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

import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import {
  Button,
  Divider,
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

const UNIT_TYPE_OPTIONS = [
  { label: 'newapi', value: 'newapi' },
  { label: 'rixapi', value: 'rixapi' },
  { label: 'shellapi', value: 'shellapi' },
  { label: 'oneapi', value: 'oneapi' },
  { label: 'oneapifork', value: 'oneapifork' },
  { label: 'veloera', value: 'veloera' },
  { label: 'onehub', value: 'onehub' },
  { label: 'donehub', value: 'donehub' },
  { label: 'anyrouter', value: 'anyrouter' },
  { label: 'sub2api', value: 'sub2api' },
  { label: 'openai', value: 'openai' },
  { label: 'claude', value: 'claude' },
  { label: 'gemini', value: 'gemini' },
  { label: 'geminicli', value: 'geminicli' },
  { label: 'antigravity', value: 'antigravity' },
  { label: 'cliproxyapi', value: 'cliproxyapi' },
  { label: 'codex', value: 'codex' },
  { label: 'unknown', value: 'unknown' },
  { label: 'other', value: 'other' },
];

const STATUS_OPTIONS = [
  { label: '启用', value: 1 },
  { label: '停用', value: 2 },
];

const DEFAULT_FORM_VALUES = {
  name: '',
  remark: '',
  website_url: '',
  api_url: '',
  type: 'newapi',
  status: 1,
};

const createEmptyAccount = () => ({
  id: 0,
  account: '',
  password: '',
  access_token: '',
  account_id: '',
});

const getEffectiveApiUrl = (unit) =>
  unit.api_url || unit.effective_api_url || unit.website_url || '';

const formatMoney = (value, unit = 'USD') => {
  const num = Number(value || 0);
  return `${unit || 'USD'} ${num.toFixed(6)}`;
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

const UnitTypeTag = ({ value }) => (
  <Tag color='blue' shape='circle'>
    {value || 'newapi'}
  </Tag>
);

const UnitStatusTag = ({ value }) => {
  const enabled = value === 1;
  return (
    <Tag color={enabled ? 'green' : 'grey'} shape='circle'>
      {enabled ? '启用' : '停用'}
    </Tag>
  );
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

const UnitsTable = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const formApiRef = useRef(null);
  const initialWebsiteURLRef = useRef('');
  const lastAutoDetectURLRef = useRef('');
  const [units, setUnits] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [modalLoading, setModalLoading] = useState(false);
  const [detectingType, setDetectingType] = useState(false);
  const [detectStatus, setDetectStatus] = useState('');
  const [websiteURL, setWebsiteURL] = useState('');
  const [editingUnit, setEditingUnit] = useState(null);
  const [accountRows, setAccountRows] = useState([]);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [keyword, setKeyword] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [accountMonitorMap, setAccountMonitorMap] = useState({});
  const [checkingAccountIds, setCheckingAccountIds] = useState({});
  const [detailMonitor, setDetailMonitor] = useState(null);
  const [detailUnit, setDetailUnit] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailMonitorMap, setDetailMonitorMap] = useState({});

  const queryUnits = useCallback(
    async (page = activePage, size = pageSize) => {
      setLoading(true);
      try {
        const res = await API.get('/api/unit/', {
          params: {
            p: page,
            page_size: size,
            keyword,
            type: typeFilter,
            status: statusFilter,
          },
        });
        const { success, message, data } = res.data;
        if (!success) {
          showError(message);
          return;
        }
        setUnits(data.items || []);
        setTotal(data.total || 0);
        setActivePage(data.page || page);
        setPageSize(data.page_size || size);
      } catch (error) {
        showError(t('获取单位列表失败'));
      } finally {
        setLoading(false);
      }
    },
    [activePage, keyword, pageSize, statusFilter, t, typeFilter],
  );

  useEffect(() => {
    queryUnits(1, pageSize);
  }, [keyword, statusFilter, typeFilter]);

  useEffect(() => {
    queryUnits(activePage, pageSize);
  }, []);

  const openCreateModal = () => {
    setEditingUnit(null);
    setAccountRows([]);
    setWebsiteURL('');
    setDetectStatus('');
    initialWebsiteURLRef.current = '';
    lastAutoDetectURLRef.current = '';
    setModalVisible(true);
    setTimeout(() => formApiRef.current?.setValues(DEFAULT_FORM_VALUES), 0);
  };

  const openEditModal = async (unit) => {
    setModalLoading(true);
    setModalVisible(true);
    try {
      const res = await API.get(`/api/unit/${unit.id}`);
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        closeModal();
        return;
      }
      setEditingUnit(data);
      setAccountRows(Array.isArray(data.accounts) ? data.accounts : []);
      loadUnitAccountMonitors(data.id).then();
      setWebsiteURL(data.website_url || '');
      setDetectStatus('');
      initialWebsiteURLRef.current = data.website_url || '';
      lastAutoDetectURLRef.current = data.website_url || '';
      setTimeout(() => {
        formApiRef.current?.setValues({
          ...DEFAULT_FORM_VALUES,
          ...data,
          type: data.type || 'newapi',
          status: data.status || 1,
        });
      }, 0);
    } catch (error) {
      showError(t('加载单位信息失败'));
      closeModal();
    } finally {
      setModalLoading(false);
    }
  };

  const closeModal = () => {
    setModalVisible(false);
    setEditingUnit(null);
    setAccountRows([]);
    setWebsiteURL('');
    setDetectStatus('');
    setDetectingType(false);
    setAccountMonitorMap({});
    setCheckingAccountIds({});
    setDetailMonitor(null);
    initialWebsiteURLRef.current = '';
    lastAutoDetectURLRef.current = '';
    formApiRef.current?.reset();
  };

  const loadUnitAccountMonitors = async (unitId) => {
    const parsedUnitId = Number(unitId || 0);
    if (!parsedUnitId) return;
    try {
      const res = await API.get('/api/unit_monitor/', {
        params: {
          p: 1,
          page_size: 500,
          unit_id: parsedUnitId,
        },
      });
      const { success, data, message } = res.data;
      if (!success) {
        showError(message || t('获取账号统计源失败'));
        return;
      }
      const nextMap = {};
      (data.items || []).forEach((monitor) => {
        nextMap[monitor.unit_account_id] = monitor;
      });
      setAccountMonitorMap(nextMap);
    } catch (error) {
      showError(t('获取账号统计源失败'));
    }
  };

  const updateAccountMonitor = (monitor) => {
    if (!monitor?.unit_account_id) return;
    setAccountMonitorMap((current) => ({
      ...current,
      [monitor.unit_account_id]: monitor,
    }));
  };

  const fetchUnitAccountMonitorsMap = async (unitId) => {
    const parsedUnitId = Number(unitId || 0);
    if (!parsedUnitId) return {};
    const res = await API.get('/api/unit_monitor/', {
      params: {
        p: 1,
        page_size: 500,
        unit_id: parsedUnitId,
      },
    });
    const { success, data, message } = res.data;
    if (!success) {
      throw new Error(message || t('获取账号统计源失败'));
    }
    const nextMap = {};
    (data.items || []).forEach((monitor) => {
      nextMap[monitor.unit_account_id] = monitor;
    });
    return nextMap;
  };

  const ensureAccountMonitor = async (unit, account, currentMap, index) => {
    if (!unit?.id || !account?.id) return null;
    if (currentMap[account.id]) {
      const monitor = currentMap[account.id];
      const res = await API.post(`/api/unit_monitor/${monitor.id}/check`);
      const { success, data, message } = res.data;
      if (!success) {
        return { ...monitor, platform_status: 'error', error_message: message || t('刷新统计源失败') };
      }
      return data;
    }
    if (!account.access_token || !account.account_id) {
      return null;
    }
    const createRes = await API.post('/api/unit_monitor/', {
      unit_id: unit.id,
      unit_account_id: account.id,
      name: `${unit.name || t('单位')} / ${account.account || account.account_id || `#${index + 1}`}`,
    });
    const { success, data, message } = createRes.data;
    if (!success) {
      throw new Error(message || t('创建统计源失败'));
    }
    const checkRes = await API.post(`/api/unit_monitor/${data.id}/check`);
    const checkPayload = checkRes.data;
    if (!checkPayload.success) {
      return {
        ...data,
        platform_status: 'error',
        error_message: checkPayload.message || t('刷新统计源失败'),
      };
    }
    return checkPayload.data;
  };

  const openDetailModal = async (unit) => {
    setDetailLoading(true);
    setDetailUnit(null);
    setDetailMonitorMap({});
    try {
      const res = await API.get(`/api/unit/${unit.id}`);
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || t('加载单位信息失败'));
        return;
      }
      setDetailUnit(data);
      let monitorMap = await fetchUnitAccountMonitorsMap(data.id);
      const accounts = Array.isArray(data.accounts) ? data.accounts : [];
      for (let index = 0; index < accounts.length; index += 1) {
        const monitor = await ensureAccountMonitor(data, accounts[index], monitorMap, index);
        if (monitor?.unit_account_id) {
          monitorMap = { ...monitorMap, [monitor.unit_account_id]: monitor };
        }
      }
      setDetailMonitorMap(monitorMap);
      setAccountMonitorMap((current) => ({ ...current, ...monitorMap }));
    } catch (error) {
      showError(error.message || t('加载单位详情失败'));
    } finally {
      setDetailLoading(false);
    }
  };

  const createAndCheckAccountMonitor = async (account, index) => {
    if (!editingUnit?.id) {
      showError(t('请先保存单位后再创建统计源'));
      return;
    }
    if (!account?.id) {
      showError(t('请先保存账号后再创建统计源'));
      return;
    }
    setCheckingAccountIds((current) => ({ ...current, [account.id]: true }));
    try {
      const createRes = await API.post('/api/unit_monitor/', {
        unit_id: editingUnit.id,
        unit_account_id: account.id,
        name: `${editingUnit.name || t('单位')} / ${account.account || account.account_id || `#${index + 1}`}`,
      });
      const { success, data, message } = createRes.data;
      if (!success) {
        showError(message || t('创建统计源失败'));
        return;
      }
      const monitor = data;
      updateAccountMonitor(monitor);
      const checkRes = await API.post(`/api/unit_monitor/${monitor.id}/check`);
      const checkPayload = checkRes.data;
      if (!checkPayload.success) {
        showError(checkPayload.message || t('刷新统计源失败'));
        await loadUnitAccountMonitors(editingUnit.id);
        return;
      }
      updateAccountMonitor(checkPayload.data);
      showSuccess(t('统计源已创建并检测'));
    } catch (error) {
      showError(t('创建统计源失败'));
    } finally {
      setCheckingAccountIds((current) => ({ ...current, [account.id]: false }));
    }
  };

  const checkAccountMonitor = async (monitor) => {
    if (!monitor?.id) return;
    setCheckingAccountIds((current) => ({
      ...current,
      [monitor.unit_account_id]: true,
    }));
    try {
      const res = await API.post(`/api/unit_monitor/${monitor.id}/check`);
      const { success, data, message } = res.data;
      if (!success) {
        showError(message || t('刷新统计源失败'));
        await loadUnitAccountMonitors(monitor.unit_id);
        return;
      }
      updateAccountMonitor(data);
      showSuccess(t('统计源已刷新'));
    } catch (error) {
      showError(t('刷新统计源失败'));
    } finally {
      setCheckingAccountIds((current) => ({
        ...current,
        [monitor.unit_account_id]: false,
      }));
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
        gridTemplateColumns: isMobile ? '1fr' : '150px minmax(0, 1fr)',
        gap: isMobile ? 6 : '10px 14px',
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

  const renderMonitorDetailSummary = (record) => {
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

  const renderPlatformBillingInfo = (monitor) => {
    const raw = parseRawJSON(monitor?.raw_json);
    const status = raw.platform_status || {};
    const generalItems = [
      { label: t('系统名称'), value: status.system_name },
      { label: t('版本'), value: status.version },
      { label: t('服务器地址'), value: status.server_address },
      { label: t('额度单位'), value: status.quota_display_type },
      { label: t('QuotaPerUnit'), value: status.quota_per_unit },
      { label: t('充值倍率 price'), value: status.price },
      { label: t('Stripe 单价'), value: status.stripe_unit_price },
      { label: t('USD 汇率'), value: status.usd_exchange_rate },
      { label: t('自定义货币符号'), value: status.custom_currency_symbol },
      { label: t('自定义货币汇率'), value: status.custom_currency_exchange_rate },
      { label: t('充值链接'), value: status.top_up_link },
    ];
    return (
      <div>
        <Text strong>{t('平台计价与充值')}</Text>
        <div style={{ marginTop: 10 }}>{renderInfoGrid(generalItems)}</div>
      </div>
    );
  };

  const renderUnitDetail = () => {
    if (!detailUnit) {
      return <Text type='secondary'>{t('正在加载单位详情...')}</Text>;
    }
    const accounts = Array.isArray(detailUnit.accounts) ? detailUnit.accounts : [];
    return (
      <div className='space-y-4'>
        <div>
          <Text strong>{t('单位信息')}</Text>
          <div style={{ marginTop: 10 }}>
            {renderInfoGrid([
              { label: 'ID', value: detailUnit.id },
              { label: t('单位名称'), value: detailUnit.name },
              { label: t('单位类型'), value: detailUnit.type },
              { label: t('平台状态'), value: detailUnit.status === 1 ? t('启用') : t('停用') },
              { label: t('单位网站地址'), value: detailUnit.website_url },
              { label: t('单位 API 地址'), value: getEffectiveApiUrl(detailUnit) },
              { label: t('挂载账号'), value: accounts.length },
              { label: t('单位备注'), value: detailUnit.remark },
              {
                label: t('更新时间'),
                value: detailUnit.updated_time ? timestamp2string(detailUnit.updated_time) : '-',
              },
            ])}
          </div>
        </div>
        <Divider margin='12px' />
        <div>
          <Text strong>{t('账号与自动采集详情')}</Text>
          <div className='flex flex-col gap-3' style={{ marginTop: 10 }}>
            {accounts.length === 0 ? (
              <Text type='secondary'>{t('暂无挂载账号')}</Text>
            ) : null}
            {accounts.map((account, index) => {
              const monitor = detailMonitorMap[account.id];
              return (
                <div
                  key={account.id || index}
                  className='p-3 rounded-md'
                  style={{
                    border: '1px solid var(--semi-color-border)',
                    background: 'var(--semi-color-fill-0)',
                  }}
                >
                  <div className='flex flex-col md:flex-row md:items-center md:justify-between gap-2'>
                    <div>
                      <Text strong>
                        {account.account || account.account_id || `#${index + 1}`}
                      </Text>
                      <div className='text-xs text-gray-500'>
                        {t('账号 ID')}：{account.account_id || '-'} · {t('访问令牌')}：
                        {account.access_token ? t('已配置') : t('未配置')}
                      </div>
                    </div>
                    <Space wrap>
                      {monitor ? <MonitorStatusTag value={monitor.platform_status} /> : null}
                      <Button
                        size='small'
                        disabled={!monitor}
                        onClick={() => setDetailMonitor(monitor)}
                      >
                        {t('账号详情')}
                      </Button>
                    </Space>
                  </div>
                  {monitor ? (
                    <div style={{ marginTop: 12 }}>
                      {renderInfoGrid([
                        {
                          label: t('余额'),
                          value: formatMoney(monitor.current_balance, monitor.balance_unit),
                        },
                        {
                          label: t('已用'),
                          value: formatMoney(monitor.used_amount, monitor.balance_unit),
                        },
                        { label: t('上游用户'), value: monitor.upstream_username },
                        { label: t('上游分组'), value: monitor.upstream_group },
                        { label: t('令牌数量'), value: monitor.token_count },
                        { label: t('模型数量'), value: monitor.model_count },
                        { label: t('分组数量'), value: monitor.group_count },
                        {
                          label: t('最后检查'),
                          value: monitor.last_checked_time
                            ? timestamp2string(monitor.last_checked_time)
                            : '-',
                        },
                        { label: t('错误信息'), value: monitor.error_message },
                      ])}
                      <div style={{ marginTop: 12 }}>
                        {renderPlatformBillingInfo(monitor)}
                      </div>
                    </div>
                  ) : (
                    <Text type='tertiary' style={{ display: 'block', marginTop: 10 }}>
                      {account.access_token && account.account_id
                        ? t('自动创建或检测统计源失败，请检查账号访问令牌。')
                        : t('缺少账户访问令牌或账户 ID，无法自动获取详情。')}
                    </Text>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      </div>
    );
  };

  const copyMonitorDetail = async () => {
    if (!detailMonitor) return;
    try {
      await navigator.clipboard.writeText(formatRawJSON(detailMonitor.raw_json));
      showSuccess(t('已复制'));
    } catch (error) {
      showInfo(t('复制失败，请手动选择复制'));
    }
  };

  const detectUnitType = useCallback(
    async (siteURL, manual = false) => {
      const normalizedSiteURL = (siteURL || '').trim();
      if (!normalizedSiteURL) {
        if (manual) {
          showError(t('请先填写单位网站地址'));
        }
        return;
      }
      setDetectingType(true);
      setDetectStatus(t('正在检测单位类型...'));
      try {
        const res = await API.post('/api/unit/detect_type', {
          site_url: normalizedSiteURL,
        });
        const { success, message, data } = res.data;
        if (!success) {
          throw new Error(message || t('检测单位类型失败'));
        }
        if (!data?.ok) {
          throw new Error(data?.message || t('检测单位类型失败'));
        }
        const resultType = data?.type || '';
        const resultMessage = data?.message || '';
        if (resultType) {
          formApiRef.current?.setValue('type', resultType);
        }
        const currentApiURL = formApiRef.current?.getValue?.('api_url') || '';
        if (data?.base_url && !currentApiURL) {
          formApiRef.current?.setValue('api_url', data.base_url);
        }
        const statusText = `${t('检测结果')}：${resultType || '-'}${
          resultMessage ? ` · ${resultMessage}` : ''
        }`;
        setDetectStatus(statusText);
        lastAutoDetectURLRef.current = normalizedSiteURL;
        if (manual) {
          showInfo(statusText);
        }
      } catch (error) {
        const message = error.message || t('检测单位类型失败');
        setDetectStatus(`${t('检测失败')}：${message}`);
        if (manual) {
          showError(message);
        }
      } finally {
        setDetectingType(false);
      }
    },
    [t],
  );

  useEffect(() => {
    if (!modalVisible) return undefined;
    const siteURL = (websiteURL || '').trim();
    if (!siteURL) return undefined;
    if (siteURL === initialWebsiteURLRef.current) return undefined;
    if (siteURL === lastAutoDetectURLRef.current) return undefined;
    const timer = setTimeout(() => {
      detectUnitType(siteURL, false);
    }, 900);
    return () => clearTimeout(timer);
  }, [detectUnitType, modalVisible, websiteURL]);

  const updateAccountRow = (index, field, value) => {
    setAccountRows((rows) =>
      rows.map((row, rowIndex) =>
        rowIndex === index ? { ...row, [field]: value } : row,
      ),
    );
  };

  const addAccountRow = () => {
    setAccountRows((rows) => [...rows, createEmptyAccount()]);
  };

  const removeAccountRow = (index) => {
    setAccountRows((rows) => rows.filter((_, rowIndex) => rowIndex !== index));
  };

  const submitUnit = async () => {
    if (!formApiRef.current) return;
    setModalLoading(true);
    try {
      const values = await formApiRef.current.validate();
      const payload = {
        ...DEFAULT_FORM_VALUES,
        ...values,
        id: editingUnit?.id,
        status: Number(values.status || 1),
        accounts: accountRows
          .map((row) => ({
            id: row.id || 0,
            account: row.account || '',
            password: row.password || '',
            access_token: row.access_token || '',
            account_id: row.account_id || '',
          }))
          .filter(
            (row) =>
              row.account || row.password || row.access_token || row.account_id,
          ),
      };
      const res = editingUnit?.id
        ? await API.put('/api/unit/', payload)
        : await API.post('/api/unit/', payload);
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(editingUnit?.id ? t('单位更新成功') : t('单位创建成功'));
      closeModal();
      queryUnits(activePage, pageSize);
    } catch (error) {
      if (error?.message) {
        showError(error.message);
      }
    } finally {
      setModalLoading(false);
    }
  };

  const updateUnitStatus = async (unit, status) => {
    const res = await API.put('/api/unit/', { ...unit, status });
    const { success, message } = res.data;
    if (!success) {
      showError(message);
      return;
    }
    showSuccess(t('状态已更新'));
    queryUnits(activePage, pageSize);
  };

  const deleteUnit = async (unit) => {
    const res = await API.delete(`/api/unit/${unit.id}`);
    const { success, message } = res.data;
    if (!success) {
      showError(message);
      return;
    }
    showSuccess(t('单位已删除'));
    queryUnits(activePage, pageSize);
  };

  const actionsArea = useMemo(
    () => (
      <div className='flex flex-col md:flex-row justify-between items-center gap-2 w-full'>
        <Space wrap>
          <Button type='primary' icon={<IconPlus />} onClick={openCreateModal}>
            {t('新建单位')}
          </Button>
          <Button
            icon={<IconRefresh />}
            onClick={() => queryUnits(activePage, pageSize)}
            loading={loading}
          >
            {t('刷新')}
          </Button>
        </Space>
        <Space wrap>
          <Input
            showClear
            prefix={<IconSearch />}
            placeholder={t('搜索单位名称 / 地址 / 备注')}
            style={{ width: isMobile ? '100%' : 260 }}
            value={keyword}
            onChange={(value) => setKeyword(value)}
          />
          <Select
            placeholder={t('单位类型')}
            showClear
            value={typeFilter || undefined}
            optionList={UNIT_TYPE_OPTIONS}
            style={{ width: 150 }}
            onChange={(value) => setTypeFilter(value || '')}
          />
          <Select
            placeholder={t('平台状态')}
            showClear
            value={statusFilter || undefined}
            optionList={STATUS_OPTIONS}
            style={{ width: 130 }}
            onChange={(value) => setStatusFilter(value || '')}
          />
        </Space>
      </div>
    ),
    [
      activePage,
      isMobile,
      keyword,
      loading,
      pageSize,
      queryUnits,
      statusFilter,
      t,
      typeFilter,
    ],
  );

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
      fixed: isMobile ? undefined : 'left',
    },
    {
      title: t('单位名称'),
      dataIndex: 'name',
      width: 200,
      render: (value) => <Text strong>{value}</Text>,
    },
    {
      title: t('单位类型'),
      dataIndex: 'type',
      width: 130,
      render: (value) => <UnitTypeTag value={value} />,
    },
    {
      title: t('平台状态'),
      dataIndex: 'status',
      width: 120,
      render: (value) => <UnitStatusTag value={value} />,
    },
    {
      title: t('单位网站地址'),
      dataIndex: 'website_url',
      width: 260,
      render: (value) =>
        value ? (
          <Text copyable={{ content: value }} ellipsis={{ showTooltip: true }}>
            {value}
          </Text>
        ) : (
          '-'
        ),
    },
    {
      title: t('单位 API 地址'),
      dataIndex: 'api_url',
      width: 280,
      render: (value, record) => {
        const effectiveApiUrl = getEffectiveApiUrl(record);
        if (!effectiveApiUrl) return '-';
        return (
          <Space spacing={6}>
            <Text
              copyable={{ content: effectiveApiUrl }}
              ellipsis={{ showTooltip: true }}
              style={{ maxWidth: 200 }}
            >
              {effectiveApiUrl}
            </Text>
            {!value ? (
              <Tag color='grey' size='small'>
                {t('继承网站地址')}
              </Tag>
            ) : null}
          </Space>
        );
      },
    },
    {
      title: t('挂载账号'),
      dataIndex: 'account_count',
      width: 110,
      render: (value) => value || 0,
    },
    {
      title: t('单位备注'),
      dataIndex: 'remark',
      width: 240,
      render: (value) =>
        value ? <Text ellipsis={{ showTooltip: true }}>{value}</Text> : '-',
    },
    {
      title: t('更新时间'),
      dataIndex: 'updated_time',
      width: 180,
      render: (value) => (value ? timestamp2string(value) : '-'),
    },
    {
      title: t('操作'),
      width: 280,
      fixed: isMobile ? undefined : 'right',
      render: (_, record) => (
        <Space>
          <Button size='small' onClick={() => openEditModal(record)}>
            {t('编辑')}
          </Button>
          <Button
            size='small'
            type={record.status === 1 ? 'warning' : 'secondary'}
            onClick={() =>
              updateUnitStatus(record, record.status === 1 ? 2 : 1)
            }
          >
            {record.status === 1 ? t('停用') : t('启用')}
          </Button>
          <Popconfirm
            title={t('确定删除该单位？')}
            content={t('删除后不可恢复。')}
            onConfirm={() => deleteUnit(record)}
          >
            <Button size='small' type='danger'>
              {t('删除')}
            </Button>
          </Popconfirm>
          <Button size='small' onClick={() => openDetailModal(record)}>
            {t('详情')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <>
      <CardPro
        type='type3'
        actionsArea={actionsArea}
        paginationArea={createCardProPagination({
          currentPage: activePage,
          pageSize,
          total,
          onPageChange: (page) => queryUnits(page, pageSize),
          onPageSizeChange: (size) => queryUnits(1, size),
          isMobile,
          t,
        })}
        t={t}
      >
        <Table
          rowKey='id'
          columns={columns}
          dataSource={units}
          loading={loading}
          pagination={false}
          scroll={{ x: 1770 }}
          empty={t('暂无单位')}
        />
      </CardPro>
      <Modal
        title={editingUnit?.id ? t('编辑单位') : t('新建单位')}
        visible={modalVisible}
        onOk={submitUnit}
        onCancel={closeModal}
        confirmLoading={modalLoading}
        okText={t('保存')}
        cancelText={t('取消')}
        width={720}
      >
        <Form
          getFormApi={(api) => {
            formApiRef.current = api;
          }}
          initValues={DEFAULT_FORM_VALUES}
          labelPosition='left'
          labelWidth={120}
        >
          <Form.Input
            field='name'
            label={t('单位名称')}
            placeholder={t('请输入单位名称')}
            rules={[{ required: true, message: t('单位名称不能为空') }]}
          />
          <Form.Input
            field='website_url'
            label={t('单位网站地址')}
            placeholder='https://example.com'
            onChange={(value) => {
              setWebsiteURL(value);
              setDetectStatus('');
            }}
          />
          <Form.Input
            field='api_url'
            label={t('单位 API 地址')}
            placeholder={t('为空时默认读取单位网站地址')}
          />
          <Form.Select
            field='type'
            label={t('单位类型')}
            optionList={UNIT_TYPE_OPTIONS}
            rules={[{ required: true, message: t('请选择单位类型') }]}
            extraText={
              <Space wrap>
                <Button
                  size='small'
                  type='primary'
                  theme='light'
                  loading={detectingType}
                  onClick={() =>
                    detectUnitType(
                      formApiRef.current?.getValue?.('website_url') ||
                        websiteURL,
                      true,
                    )
                  }
                >
                  {t('检测类型')}
                </Button>
                {detectStatus ? (
                  <Text
                    type={
                      detectStatus.includes(t('检测结果'))
                        ? 'success'
                        : 'danger'
                    }
                    size='small'
                  >
                    {detectStatus}
                  </Text>
                ) : null}
              </Space>
            }
          />
          <Form.Select
            field='status'
            label={t('平台状态')}
            optionList={STATUS_OPTIONS}
            rules={[{ required: true, message: t('请选择平台状态') }]}
          />
          <Form.TextArea
            field='remark'
            label={t('单位备注')}
            placeholder={t('请输入单位备注')}
            autosize={{ minRows: 3, maxRows: 6 }}
          />
        </Form>
        <Divider margin='16px' />
        <div className='flex justify-between items-center mb-3'>
          <Text strong>{t('挂载账号')}</Text>
          <Button size='small' icon={<IconPlus />} onClick={addAccountRow}>
            {t('新增账号')}
          </Button>
        </div>
        <div className='flex flex-col gap-3'>
          {accountRows.length === 0 ? (
            <Text type='secondary'>{t('暂无挂载账号')}</Text>
          ) : null}
          {accountRows.map((row, index) => {
            const monitor = row.id ? accountMonitorMap[row.id] : null;
            const checking = row.id ? !!checkingAccountIds[row.id] : false;
            return (
              <div
                key={`${row.id || 'new'}-${index}`}
                className='grid grid-cols-1 md:grid-cols-2 gap-3 p-3 rounded-md'
                style={{
                  border: '1px solid var(--semi-color-border)',
                  background: 'var(--semi-color-fill-0)',
                }}
              >
                <Input
                  placeholder={t('账号')}
                  value={row.account}
                  onChange={(value) =>
                    updateAccountRow(index, 'account', value)
                  }
                />
                <Input
                  placeholder={t('密码')}
                  value={row.password}
                  onChange={(value) =>
                    updateAccountRow(index, 'password', value)
                  }
                />
                <Input
                  placeholder={t('账户访问令牌')}
                  value={row.access_token}
                  onChange={(value) =>
                    updateAccountRow(index, 'access_token', value)
                  }
                />
                <Space>
                  <Input
                    placeholder={t('账户 ID')}
                    value={row.account_id}
                    onChange={(value) =>
                      updateAccountRow(index, 'account_id', value)
                    }
                  />
                  <Button
                    type='danger'
                    theme='borderless'
                    onClick={() => removeAccountRow(index)}
                  >
                    {t('删除')}
                  </Button>
                </Space>
                <div
                  className='md:col-span-2 flex flex-col md:flex-row md:items-center md:justify-between gap-2'
                  style={{
                    borderTop: '1px solid var(--semi-color-border)',
                    paddingTop: 10,
                  }}
                >
                  {monitor ? (
                    <Space wrap>
                      <MonitorStatusTag value={monitor.platform_status} />
                      <Text type='secondary'>
                        {t('余额')}：
                        {formatMoney(
                          monitor.current_balance,
                          monitor.balance_unit,
                        )}
                      </Text>
                      <Text type='secondary'>
                        {t('已用')}：
                        {formatMoney(monitor.used_amount, monitor.balance_unit)}
                      </Text>
                      <Text type='secondary'>
                        {t('最后检查')}：
                        {monitor.last_checked_time
                          ? timestamp2string(monitor.last_checked_time)
                          : '-'}
                      </Text>
                    </Space>
                  ) : (
                    <Text type='tertiary'>
                      {row.id
                        ? t('该账号尚未创建统计源')
                        : t('保存账号后可创建统计源')}
                    </Text>
                  )}
                  <Space>
                    {monitor ? (
                      <>
                        <Button
                          size='small'
                          icon={<IconRefresh />}
                          loading={checking}
                          onClick={() => checkAccountMonitor(monitor)}
                        >
                          {t('刷新')}
                        </Button>
                        <Button
                          size='small'
                          onClick={() => setDetailMonitor(monitor)}
                        >
                          {t('详情')}
                        </Button>
                      </>
                    ) : (
                      <Button
                        size='small'
                        type='primary'
                        theme='light'
                        loading={checking}
                        disabled={!row.id}
                        onClick={() => createAndCheckAccountMonitor(row, index)}
                      >
                        {t('创建统计源并检测')}
                      </Button>
                    )}
                  </Space>
                </div>
              </div>
            );
          })}
        </div>
      </Modal>
      <Modal
        title={t('单位详情')}
        visible={detailLoading || !!detailUnit}
        onCancel={() => {
          setDetailUnit(null);
          setDetailMonitorMap({});
          setDetailLoading(false);
        }}
        footer={
          <Button
            onClick={() => {
              setDetailUnit(null);
              setDetailMonitorMap({});
              setDetailLoading(false);
            }}
          >
            {t('关闭')}
          </Button>
        }
        width={860}
      >
        {detailLoading ? (
          <Text type='secondary'>{t('正在自动获取单位详情...')}</Text>
        ) : (
          renderUnitDetail()
        )}
      </Modal>
      <Modal
        title={t('单位账号统计详情')}
        visible={!!detailMonitor}
        onCancel={() => setDetailMonitor(null)}
        footer={
          <Space>
            <Button onClick={() => setDetailMonitor(null)}>{t('关闭')}</Button>
            <Button type='primary' onClick={copyMonitorDetail}>
              {t('复制原始信息')}
            </Button>
          </Space>
        }
        width={760}
      >
        {detailMonitor ? (
          <div>
            <div>
              <Text strong>{detailMonitor.name || '-'}</Text>
              <div className='text-xs text-gray-500'>
                {detailMonitor.unit_name || '-'} · {detailMonitor.account || '-'}
              </div>
            </div>
            <Tabs defaultActiveKey='summary' size='medium' style={{ marginTop: 16 }}>
              <Tabs.TabPane tab={t('摘要')} itemKey='summary'>
                {renderMonitorDetailSummary(detailMonitor)}
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
                  {formatRawJSON(detailMonitor.raw_json)}
                </pre>
              </Tabs.TabPane>
            </Tabs>
          </div>
        ) : null}
      </Modal>
    </>
  );
};

export default UnitsTable;
