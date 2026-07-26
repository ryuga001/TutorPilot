import { createApi } from "@reduxjs/toolkit/query/react";

import { baseQueryWithReauth } from "@/lib/api/baseQuery";
import type {
  CreateStudentInput,
  Envelope,
  Paginated,
  Student,
  UpdateStudentInput,
} from "@/lib/types";

interface ListArgs {
  page?: number;
  page_size?: number;
  search?: string;
}

export const studentsApi = createApi({
  reducerPath: "studentsApi",
  baseQuery: baseQueryWithReauth,
  tagTypes: ["Student"],
  endpoints: (build) => ({
    listStudents: build.query<Paginated<Student>, ListArgs>({
      query: ({ page = 1, page_size = 20, search = "" }) => ({
        url: "/students",
        params: { page, page_size, search },
      }),
      transformResponse: (r: Envelope<Paginated<Student>>) =>
        r.data as Paginated<Student>,
      providesTags: ["Student"],
    }),

    createStudent: build.mutation<Student, CreateStudentInput>({
      query: (body) => ({ url: "/students", method: "POST", body }),
      transformResponse: (r: Envelope<Student>) => r.data as Student,
      invalidatesTags: ["Student"],
    }),

    updateStudent: build.mutation<
      Student,
      { id: number; body: UpdateStudentInput }
    >({
      query: ({ id, body }) => ({ url: `/students/${id}`, method: "PUT", body }),
      transformResponse: (r: Envelope<Student>) => r.data as Student,
      invalidatesTags: ["Student"],
    }),

    deleteStudent: build.mutation<void, number>({
      query: (id) => ({ url: `/students/${id}`, method: "DELETE" }),
      invalidatesTags: ["Student"],
    }),

    uploadStudentProfileImage: build.mutation<
      Student,
      { id: number; file: File }
    >({
      query: ({ id, file }) => {
        const form = new FormData();
        form.append("file", file);
        return {
          url: `/students/${id}/profile-image`,
          method: "POST",
          body: form,
        };
      },
      transformResponse: (r: Envelope<Student>) => r.data as Student,
      invalidatesTags: ["Student"],
    }),
  }),
});

export const {
  useListStudentsQuery,
  useCreateStudentMutation,
  useUpdateStudentMutation,
  useDeleteStudentMutation,
  useUploadStudentProfileImageMutation,
} = studentsApi;
