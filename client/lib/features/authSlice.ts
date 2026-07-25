import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import type { AuthResponse, TokenPair, UserView } from "@/lib/types";

export interface AuthState {
  user: UserView | null;
  accessToken: string | null;
  refreshToken: string | null;
  privileges: string[] | null;
}

export const initialAuthState: AuthState = {
  user: null,
  accessToken: null,
  refreshToken: null,
  privileges: null,
};

const authSlice = createSlice({
  name: "auth",
  initialState: initialAuthState,
  reducers: {
    setCredentials(state, action: PayloadAction<AuthResponse>) {
      state.user = action.payload.user;
      state.accessToken = action.payload.tokens.access_token;
      state.refreshToken = action.payload.tokens.refresh_token;
      state.privileges = null;
    },
    setPrivileges(state, action: PayloadAction<string[]>) {
      state.privileges = action.payload;
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
      state.privileges = null;
    },
  },
});

export const { setCredentials, setTokens, setUser, setPrivileges, logout } =
  authSlice.actions;
export default authSlice.reducer;
