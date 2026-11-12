import axios, {
  AxiosError,
  AxiosInstance,
  AxiosRequestConfig,
  AxiosResponse,
} from 'axios';
import { message } from '@ctzhian/ui';
import { clearAuth, getLegacyBearerToken } from '@/utils/auth';

type BasicResponse<T> = {
  data: T;
  success: boolean;
  message: string;
};

type ErrorResponse = {
  data: unknown;
  success: boolean;
  message: string;
};

type Response<T> = BasicResponse<T> | ErrorResponse;

const request = <T>(options: AxiosRequestConfig): Promise<T> => {
  const token = getLegacyBearerToken();
  const headers: Record<string, string> = {};
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  const config = {
    baseURL: window.__BASENAME__ || '/',
    timeout: 0,
    withCredentials: true,
    headers,
  };
  const service: AxiosInstance = axios.create(config);
  service.interceptors.response.use(
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    (response: AxiosResponse<Response<T>>) => {
      if (response.status === 200) {
        const res = response.data;
        if (res.success) {
          return res.data;
        }
        message.error(res.message || '网络异常');
        return Promise.reject(res);
      }
      message.error(response.statusText);
      return Promise.reject(response);
    },
    (error: AxiosError) => {
      if (error.response?.status === 401) {
        clearAuth();
        window.location.href = window.__BASENAME__ + '/login';
      }
      message.error(error.response?.statusText || '网络异常');
      return Promise.reject(error.response);
    },
  );

  return service(options);
};

export default request;
