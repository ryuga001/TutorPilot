import {
  fetchBaseQuery,
  type BaseQueryFn,
  type FetchArgs,
  type FetchBaseQueryError,
} from "@reduxjs/toolkit/query";
import { logout, setTokens } from "@/lib/features/authSlice";
import type { Envelope, TokenPair } from "@/lib/types";
import type { RootState } from "@/lib/store";

// The API host has to mirror the browser host, because the server resolves the
// tenant from the subdomain: a request from acme.localhost:3000 must reach
// acme.localhost:8080, or it arrives on the root host with no organization and
// only staff and tutors can sign in.
//
// NEXT_PUBLIC_API_URL still overrides everything, for deployments where the API
// lives on a fixed host.
const API_PORT = process.env.NEXT_PUBLIC_API_PORT ?? "8080";

function apiBaseUrl(): string {
  if (process.env.NEXT_PUBLIC_API_URL) return process.env.NEXT_PUBLIC_API_URL;
  if (typeof window === "undefined") return `http://localhost:${API_PORT}/api/v1`;

  const { protocol, hostname } = window.location;
  return `${protocol}//${hostname}:${API_PORT}/api/v1`;
}

// Built on first use rather than at module load, so window.location is available.
let baseQuery: ReturnType<typeof fetchBaseQuery> | null = null;

const rawBaseQuery: ReturnType<typeof fetchBaseQuery> = (args, api, extraOptions) => {
  baseQuery ??= fetchBaseQuery({
    baseUrl: apiBaseUrl(),
    prepareHeaders: (headers, { getState }) => {
      const token = (getState() as RootState).auth.accessToken;
      if (token) headers.set("authorization", `Bearer ${token}`);
      return headers;
    },
  });
  return baseQuery(args, api, extraOptions);
};

// Single in-flight refresh shared across concurrent 401s.
let refreshPromise: ReturnType<typeof rawBaseQuery> | null = null;

export const baseQueryWithReauth: BaseQueryFn<
  string | FetchArgs,
  unknown,
  FetchBaseQueryError
> = async (args, api, extraOptions) => {
  let result = await rawBaseQuery(args, api, extraOptions);

  if (result.error?.status === 401) {
    const refreshToken = (api.getState() as RootState).auth.refreshToken;
    if (!refreshToken) {
      api.dispatch(logout());
      return result;
    }

    if (!refreshPromise) {
      refreshPromise = rawBaseQuery(
        {
          url: "/auth/refresh",
          method: "POST",
          body: { refresh_token: refreshToken },
        },
        api,
        extraOptions,
      );
    }
    const refreshResult = await refreshPromise;
    refreshPromise = null;

    const env = refreshResult.data as Envelope<TokenPair> | undefined;
    if (env?.data) {
      api.dispatch(setTokens(env.data));
      result = await rawBaseQuery(args, api, extraOptions);
    } else {
      api.dispatch(logout());
    }
  }

  return result;
};
