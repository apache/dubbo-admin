export type TableListItem = {
  key: number;
  disabled?: boolean;
  desc: string;
  callNo: number;
  status: number;
  progress: number;
  service?: string;
  appName?: string;
  group?: string;
  version?: string;
};

export type TableListPagination = {
  total: number;
  size: number;
  page: number;
};

export type TableListData = {
  list: TableListItem[];
  pagination: Partial<TableListPagination>;
};

export type TableListParams = {
  status?: string;
  service?: string;
  appName?: string;
  group?: string;
  version?: string;
  filter?: string;
  pattern?: string;
  desc?: string;
  size?: number;
  page?: number;
  sorter?: Record<string, any>;
};
