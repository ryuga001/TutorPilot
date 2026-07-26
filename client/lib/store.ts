import { configureStore } from "@reduxjs/toolkit";
import { authApi } from "@/lib/api/authApi";
import { batchesApi } from "@/lib/api/batchesApi";
import { coursesApi } from "@/lib/api/coursesApi";
import { lecturesApi } from "@/lib/api/lecturesApi";
import { studentsApi } from "@/lib/api/studentsApi";
import { tutorsApi } from "@/lib/api/tutorsApi";
import authReducer, {
  initialAuthState,
  type AuthState,
} from "@/lib/features/authSlice";

const AUTH_STORAGE_KEY = "tutorpilot.auth";

function loadAuthState(): AuthState {
  if (typeof window === "undefined") return initialAuthState;
  try {
    const raw = window.localStorage.getItem(AUTH_STORAGE_KEY);
    if (raw) return { ...initialAuthState, ...JSON.parse(raw) };
  } catch {
    /* ignore corrupt storage */
  }
  return initialAuthState;
}

export const makeStore = () => {
  const store = configureStore({
    reducer: {
      auth: authReducer,
      [authApi.reducerPath]: authApi.reducer,
      [coursesApi.reducerPath]: coursesApi.reducer,
      [lecturesApi.reducerPath]: lecturesApi.reducer,
      [tutorsApi.reducerPath]: tutorsApi.reducer,
      [studentsApi.reducerPath]: studentsApi.reducer,
      [batchesApi.reducerPath]: batchesApi.reducer,
    },
    middleware: (getDefaultMiddleware) =>
      getDefaultMiddleware().concat(
        authApi.middleware,
        coursesApi.middleware,
        lecturesApi.middleware,
        tutorsApi.middleware,
        studentsApi.middleware,
        batchesApi.middleware,
      ),
    preloadedState: { auth: loadAuthState() },
  });

  if (typeof window !== "undefined") {
    store.subscribe(() => {
      try {
        window.localStorage.setItem(
          AUTH_STORAGE_KEY,
          JSON.stringify(store.getState().auth),
        );
      } catch {
        /* ignore quota errors */
      }
    });
  }

  return store;
};

export type AppStore = ReturnType<typeof makeStore>;
export type RootState = ReturnType<AppStore["getState"]>;
export type AppDispatch = AppStore["dispatch"];
