import { createApi } from "@reduxjs/toolkit/query/react";

import { baseQueryWithReauth } from "@/lib/api/baseQuery";
import type {
  CreateTutorInput,
  Envelope,
  Paginated,
  Tutor,
  UpdateTutorInput,
} from "@/lib/types";

interface ListArgs {
  page?: number;
  page_size?: number;
  search?: string;
}

export const tutorsApi = createApi({
  reducerPath: "tutorsApi",
  baseQuery: baseQueryWithReauth,
  tagTypes: ["Tutor"],
  endpoints: (build) => ({
    listTutors: build.query<Paginated<Tutor>, ListArgs>({
      query: ({ page = 1, page_size = 20, search = "" }) => ({
        url: "/tutors",
        params: { page, page_size, search },
      }),
      transformResponse: (r: Envelope<Paginated<Tutor>>) =>
        r.data as Paginated<Tutor>,
      providesTags: ["Tutor"],
    }),

    createTutor: build.mutation<Tutor, CreateTutorInput>({
      query: (body) => ({ url: "/tutors", method: "POST", body }),
      transformResponse: (r: Envelope<Tutor>) => r.data as Tutor,
      invalidatesTags: ["Tutor"],
    }),

    updateTutor: build.mutation<Tutor, { id: number; body: UpdateTutorInput }>({
      query: ({ id, body }) => ({ url: `/tutors/${id}`, method: "PUT", body }),
      transformResponse: (r: Envelope<Tutor>) => r.data as Tutor,
      invalidatesTags: ["Tutor"],
    }),

    deleteTutor: build.mutation<void, number>({
      query: (id) => ({ url: `/tutors/${id}`, method: "DELETE" }),
      invalidatesTags: ["Tutor"],
    }),

    uploadTutorProfileImage: build.mutation<Tutor, { id: number; file: File }>({
      query: ({ id, file }) => {
        const form = new FormData();
        form.append("file", file);
        return { url: `/tutors/${id}/profile-image`, method: "POST", body: form };
      },
      transformResponse: (r: Envelope<Tutor>) => r.data as Tutor,
      invalidatesTags: ["Tutor"],
    }),
  }),
});

export const {
  useListTutorsQuery,
  useCreateTutorMutation,
  useUpdateTutorMutation,
  useDeleteTutorMutation,
  useUploadTutorProfileImageMutation,
} = tutorsApi;
