// Feature: request-trace-view, Property 9: Quota 金额格式化
// **Validates: Requirements 6.4**

import { describe, it, expect } from 'vitest';
import fc from 'fast-check';
import { formatQuotaToUSD } from '../utils';

describe('formatQuotaToUSD - Property 9: Quota 金额格式化', () => {
  it('should format any non-negative integer quota as $X.XXXXXX (6 decimal places)', () => {
    fc.assert(
      fc.property(fc.nat(), (quota) => {
        const result = formatQuotaToUSD(quota);
        const expected = `$${(quota * 0.000001).toFixed(6)}`;
        expect(result).toBe(expected);
      }),
      { numRuns: 100 },
    );
  });
});
