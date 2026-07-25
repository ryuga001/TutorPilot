// Mirrors the Go API's httpx.Envelope and auth DTOs.

export interface Envelope<T> {
  success: boolean;
  message?: string;
  data?: T;
  error?: string;
}

export interface UserView {
  id: number;
  customer_id: number;
  email: string;
  role: string;
  created_at: string;
  first_name: string;
  last_name: string;
}

export interface Privilages {
  
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
}

export interface AuthResponse {
  user: UserView;
  tokens: TokenPair;
}

// Request payloads
export interface LoginInput {
  email: string;
  password: string;
}

export interface RegisterInput {
  org_name: string;
  first_name: string;
  last_name: string;
  email: string;
  password: string;
}

export interface VerifyEmailInput {
  email: string;
  otp: string;
}

export interface ResetPasswordInput {
  email: string;
  otp: string;
  new_password: string;
}
