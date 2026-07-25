import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import type { AuthResponse, TokenPair, UserView } from "@/lib/types";

export interface AuthState {
  user: UserView | null;
  accessToken: string | null;
  refreshToken: string | null;
}

export const initialAuthState: AuthState = {
  user: null,
  accessToken: null,
  refreshToken: null,
};

const authSlice = createSlice({
  name: "auth",
  initialState: initialAuthState,
  reducers: {
    setCredentials(state, action: PayloadAction<AuthResponse>) {
      state.user = action.payload.user;
      state.accessToken = action.payload.tokens.access_token;
      state.refreshToken = action.payload.tokens.refresh_token;
    },
    setTokens(state, action: PayloadAction<TokenPair>) {
      state.accessToken = action.payload.access_token;
      state.refreshToken = action.payload.refresh_token;
    },
    setUser(state, action: PayloadAction<UserView | null>) {
      state.user = action.payload;
    },
    logout(state) {
      state.user = null;
      state.accessToken = null;
      state.refreshToken = null;
    },
  },
});

export const { setCredentials, setTokens, setUser, logout } = authSlice.actions;
export default authSlice.reducer;
