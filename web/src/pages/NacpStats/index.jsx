import React from 'react';
import NacpStatsTable from '../../components/table/nacp-stats';
import { useIsMobile } from '../../hooks/common/useIsMobile';

const NacpStats = () => {
  const isMobile = useIsMobile();

  return (
    <div
      style={{
        paddingTop: isMobile ? 72 : 64,
      }}
    >
      <NacpStatsTable />
    </div>
  );
};

export default NacpStats;
