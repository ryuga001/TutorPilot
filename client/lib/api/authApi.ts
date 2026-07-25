import { createApi } from "@reduxjs/toolkit/query/react";
import { baseQueryWithReauth } from "@/lib/api/baseQuery";
import type {
  AuthResponse,
  Envelope,
  LoginInput,
  RegisterInput,
  ResetPasswordInput,
  UserView,
  VerifyEmailInput,
} from "@/lib/types";

type MessageResponse = Envelope<null>;

export const authApi = createApi({
  reducerPath: "authApi",
  baseQuery: baseQueryWithReauth,
  tagTypes: ["Me"],
  endpoints: (build) => ({
    login: build.mutation<AuthResponse, LoginInput>({
      query: (body) => ({ url: "/auth/login", method: "POST", body }),
      transformResponse: (r: Envelope<AuthResponse>) => r.data as AuthResponse,
      invalidatesTags: ["Me"],
    }),

    register: build.mutation<AuthResponse, RegisterInput>({
      query: (body) => ({ url: "/auth/register", method: "POST", body }),
      transformResponse: (r: Envelope<AuthResponse>) => r.data as AuthResponse,
      invalidatesTags: ["Me"],
    }),

    sendVerification: build.mutation<MessageResponse, { email: string }>({
      query: (body) => ({
        url: "/auth/send-verification",
        method: "POST",
        body,
      }),
    }),

    verifyEmail: build.mutation<MessageResponse, VerifyEmailInput>({
      query: (body) => ({ url: "/auth/verify-email", method: "POST", body }),
    }),

    forgotPassword: build.mutation<MessageResponse, { email: string }>({
      query: (body) => ({ url: "/auth/forgot-password", method: "POST", body }),
    }),

    resetPassword: build.mutation<MessageResponse, ResetPasswordInput>({
      query: (body) => ({ url: "/auth/reset-password", method: "POST", body }),
    }),

    logout: build.mutation<MessageResponse, { refresh_token: string }>({
      query: (body) => ({ url: "/auth/logout", method: "POST", body }),
      invalidatesTags: ["Me"],
    }),

    getMe: build.query<UserView, void>({
      query: () => "/auth/me",
      transformResponse: (r: Envelope<UserView>) => r.data as UserView,
      providesTags: ["Me"],
    }),
  }),
});

export const {
  useLoginMutation,
  useRegisterMutation,
  useSendVerificationMutation,
  useVerifyEmailMutation,
  useForgotPasswordMutation,
  useResetPasswordMutation,
  useLogoutMutation,
  useGetMeQuery,
} = authApi;
