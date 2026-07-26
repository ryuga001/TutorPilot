import { createApi } from "@reduxjs/toolkit/query/react";

import { baseQueryWithReauth } from "@/lib/api/baseQuery";
import type {
  CreateLectureInput,
  Envelope,
  Lecture,
  LectureJoinResponse,
  Paginated,
  UpdateLectureInput,
} from "@/lib/types";

interface ListArgs {
  page?: number;
  page_size?: number;
  search?: string;
  status?: string;
  batchId?: number;
}

export const lecturesApi = createApi({
  reducerPath: "lecturesApi",
  baseQuery: baseQueryWithReauth,
  tagTypes: ["Lecture", "LectureList"],
  endpoints: (build) => ({
    listLectures: build.query<Paginated<Lecture>, ListArgs>({
      query: ({ page = 1, page_size = 10, search = "", status = "", batchId } = {}) => ({
        url: "/lectures",
        params: { page, page_size, search, status, batchId },
      }),
      transformResponse: (r: Envelope<Paginated<Lecture>>) => r.data as Paginated<Lecture>,
      providesTags: ["LectureList"],
    }),

    getLecture: build.query<Lecture, number>({
      query: (id) => `/lectures/${id}`,
      transformResponse: (r: Envelope<Lecture>) => r.data as Lecture,
      providesTags: (_res, _err, id) => [{ type: "Lecture", id }],
    }),

    createLecture: build.mutation<Lecture, CreateLectureInput>({
      query: (body) => ({ url: "/lectures", method: "POST", body }),
      transformResponse: (r: Envelope<Lecture>) => r.data as Lecture,
      invalidatesTags: ["LectureList"],
    }),

    updateLecture: build.mutation<Lecture, { id: number; body: UpdateLectureInput }>({
      query: ({ id, body }) => ({ url: `/lectures/${id}`, method: "PUT", body }),
      transformResponse: (r: Envelope<Lecture>) => r.data as Lecture,
      invalidatesTags: (_res, _err, { id }) => [{ type: "Lecture", id }, "LectureList"],
    }),

    deleteLecture: build.mutation<void, number>({
      query: (id) => ({ url: `/lectures/${id}`, method: "DELETE" }),
      invalidatesTags: ["LectureList"],
    }),

    startLecture: build.mutation<Lecture, number>({
      query: (id) => ({ url: `/lectures/${id}/start`, method: "POST" }),
      transformResponse: (r: Envelope<Lecture>) => r.data as Lecture,
      invalidatesTags: (_res, _err, id) => [{ type: "Lecture", id }, "LectureList"],
    }),

    endLecture: build.mutation<Lecture, number>({
      query: (id) => ({ url: `/lectures/${id}/end`, method: "POST" }),
      transformResponse: (r: Envelope<Lecture>) => r.data as Lecture,
      invalidatesTags: (_res, _err, id) => [{ type: "Lecture", id }, "LectureList"],
    }),

    joinLecture: build.mutation<LectureJoinResponse, number>({
      query: (id) => ({ url: `/lectures/${id}/join`, method: "POST" }),
      transformResponse: (r: Envelope<LectureJoinResponse>) => r.data as LectureJoinResponse,
    }),
  }),
});

export const {
  useListLecturesQuery,
  useGetLectureQuery,
  useCreateLectureMutation,
  useUpdateLectureMutation,
  useDeleteLectureMutation,
  useStartLectureMutation,
  useEndLectureMutation,
  useJoinLectureMutation,
} = lecturesApi;
