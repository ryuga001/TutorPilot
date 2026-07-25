import { createApi } from "@reduxjs/toolkit/query/react";

import { baseQueryWithReauth } from "@/lib/api/baseQuery";
import type {
  Course,
  CourseModule,
  CourseLesson,
  CourseResource,
  CreateCourseInput,
  Envelope,
  LessonInput,
  ModuleInput,
  Paginated,
  UpdateCourseInput,
} from "@/lib/types";

interface ListArgs {
  page?: number;
  page_size?: number;
  search?: string;
  status?: string;
}

export const coursesApi = createApi({
  reducerPath: "coursesApi",
  baseQuery: baseQueryWithReauth,
  tagTypes: ["Course", "CourseList", "Resource"],
  endpoints: (build) => ({
    listCourses: build.query<Paginated<Course>, ListArgs>({
      query: ({ page = 1, page_size = 12, search = "", status = "" }) => ({
        url: "/courses",
        params: { page, page_size, search, status },
      }),
      transformResponse: (r: Envelope<Paginated<Course>>) =>
        r.data as Paginated<Course>,
      providesTags: ["CourseList"],
    }),

    getCourse: build.query<Course, number>({
      query: (id) => `/courses/${id}`,
      transformResponse: (r: Envelope<Course>) => r.data as Course,
      providesTags: (_res, _err, id) => [{ type: "Course", id }],
    }),

    createCourse: build.mutation<Course, CreateCourseInput>({
      query: (body) => ({ url: "/courses", method: "POST", body }),
      transformResponse: (r: Envelope<Course>) => r.data as Course,
      invalidatesTags: ["CourseList"],
    }),

    updateCourse: build.mutation<Course, { id: number; body: UpdateCourseInput }>({
      query: ({ id, body }) => ({ url: `/courses/${id}`, method: "PUT", body }),
      transformResponse: (r: Envelope<Course>) => r.data as Course,
      invalidatesTags: (_res, _err, { id }) => [{ type: "Course", id }, "CourseList"],
    }),

    deleteCourse: build.mutation<void, number>({
      query: (id) => ({ url: `/courses/${id}`, method: "DELETE" }),
      invalidatesTags: ["CourseList"],
    }),

    setCoursePublished: build.mutation<Course, { id: number; published: boolean }>({
      query: ({ id, published }) => ({
        url: `/courses/${id}/${published ? "publish" : "unpublish"}`,
        method: "POST",
      }),
      transformResponse: (r: Envelope<Course>) => r.data as Course,
      invalidatesTags: (_res, _err, { id }) => [{ type: "Course", id }, "CourseList"],
    }),

    uploadThumbnail: build.mutation<Course, { id: number; file: File }>({
      query: ({ id, file }) => {
        const form = new FormData();
        form.append("file", file);
        return { url: `/courses/${id}/thumbnail`, method: "POST", body: form };
      },
      transformResponse: (r: Envelope<Course>) => r.data as Course,
      invalidatesTags: (_res, _err, { id }) => [{ type: "Course", id }, "CourseList"],
    }),

    // --- Modules ---
    createModule: build.mutation<CourseModule, { courseId: number; body: ModuleInput }>({
      query: ({ courseId, body }) => ({
        url: `/courses/${courseId}/modules`,
        method: "POST",
        body,
      }),
      transformResponse: (r: Envelope<CourseModule>) => r.data as CourseModule,
      invalidatesTags: (_res, _err, { courseId }) => [{ type: "Course", id: courseId }],
    }),
    updateModule: build.mutation<void, { courseId: number; moduleId: number; body: ModuleInput }>({
      query: ({ courseId, moduleId, body }) => ({
        url: `/courses/${courseId}/modules/${moduleId}`,
        method: "PUT",
        body,
      }),
      invalidatesTags: (_res, _err, { courseId }) => [{ type: "Course", id: courseId }],
    }),
    deleteModule: build.mutation<void, { courseId: number; moduleId: number }>({
      query: ({ courseId, moduleId }) => ({
        url: `/courses/${courseId}/modules/${moduleId}`,
        method: "DELETE",
      }),
      invalidatesTags: (_res, _err, { courseId }) => [{ type: "Course", id: courseId }],
    }),

    // --- Lessons ---
    createLesson: build.mutation<
      CourseLesson,
      { courseId: number; moduleId: number; body: LessonInput }
    >({
      query: ({ courseId, moduleId, body }) => ({
        url: `/courses/${courseId}/modules/${moduleId}/lessons`,
        method: "POST",
        body,
      }),
      transformResponse: (r: Envelope<CourseLesson>) => r.data as CourseLesson,
      invalidatesTags: (_res, _err, { courseId }) => [{ type: "Course", id: courseId }],
    }),
    updateLesson: build.mutation<
      void,
      { courseId: number; moduleId: number; lessonId: number; body: LessonInput }
    >({
      query: ({ courseId, moduleId, lessonId, body }) => ({
        url: `/courses/${courseId}/modules/${moduleId}/lessons/${lessonId}`,
        method: "PUT",
        body,
      }),
      invalidatesTags: (_res, _err, { courseId }) => [{ type: "Course", id: courseId }],
    }),
    deleteLesson: build.mutation<
      void,
      { courseId: number; moduleId: number; lessonId: number }
    >({
      query: ({ courseId, moduleId, lessonId }) => ({
        url: `/courses/${courseId}/modules/${moduleId}/lessons/${lessonId}`,
        method: "DELETE",
      }),
      invalidatesTags: (_res, _err, { courseId }) => [{ type: "Course", id: courseId }],
    }),

    // --- Resources ---
    listResources: build.query<
      Paginated<CourseResource>,
      { courseId: number; page?: number; page_size?: number }
    >({
      query: ({ courseId, page = 1, page_size = 50 }) => ({
        url: `/courses/${courseId}/resources`,
        params: { page, page_size },
      }),
      transformResponse: (r: Envelope<Paginated<CourseResource>>) =>
        r.data as Paginated<CourseResource>,
      providesTags: ["Resource"],
    }),
    uploadResource: build.mutation<
      CourseResource,
      { courseId: number; file: File; lessonId?: number }
    >({
      query: ({ courseId, file, lessonId }) => {
        const form = new FormData();
        form.append("file", file);
        if (lessonId) form.append("lesson_id", String(lessonId));
        return { url: `/courses/${courseId}/resources`, method: "POST", body: form };
      },
      transformResponse: (r: Envelope<CourseResource>) => r.data as CourseResource,
      invalidatesTags: ["Resource"],
    }),
    deleteResource: build.mutation<void, { courseId: number; resourceId: number }>({
      query: ({ courseId, resourceId }) => ({
        url: `/courses/${courseId}/resources/${resourceId}`,
        method: "DELETE",
      }),
      invalidatesTags: ["Resource"],
    }),
  }),
});

export const {
  useListCoursesQuery,
  useGetCourseQuery,
  useCreateCourseMutation,
  useUpdateCourseMutation,
  useDeleteCourseMutation,
  useSetCoursePublishedMutation,
  useUploadThumbnailMutation,
  useCreateModuleMutation,
  useUpdateModuleMutation,
  useDeleteModuleMutation,
  useCreateLessonMutation,
  useUpdateLessonMutation,
  useDeleteLessonMutation,
  useListResourcesQuery,
  useUploadResourceMutation,
  useDeleteResourceMutation,
} = coursesApi;
