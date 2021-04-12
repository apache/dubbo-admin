import { request } from 'umi';
import type { TableListParams} from './data.d';

export async function queryService(params?: TableListParams) {
  return request('/service', {
    params,
  });
}
