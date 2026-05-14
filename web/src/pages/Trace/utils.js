/**
 * 格式化 quota 为美元金额
 * quota × 0.000001 转美元，显示 $X.XXXXXX（6 位小数）
 */
export function formatQuotaToUSD(quota) {
  const amount = quota * 0.000001;
  return `$${amount.toFixed(6)}`;
}
