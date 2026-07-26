import { createApi } from "@reduxjs/toolkit/query/react";

import { baseQueryWithReauth } from "@/lib/api/baseQuery";
import type {
  AssignTutorInput,
  Batch,
  CreateBatchInput,
  DriveNode,
  EnrollResult,
  Envelope,
  ImportResult,
  Paginated,
  StudentSummary,
  TutorSummary,
  UpdateBatchInput,
} from "@/lib/types";

interface ListBatchesArgs {
  page?: number;
  page_size?: number;
  course_id?: number;
  status?: string;
  search?: string;
}

export const batchesApi = createApi({
  reducerPath: "batchesApi",
  baseQuery: baseQueryWithReauth,
  tagTypes: ["Batch", "BatchList", "BatchTutors", "BatchStudents", "Drive"],
  endpoints: (build) => ({
    listBatches: build.query<Paginated<Batch>, ListBatchesArgs>({
      query: ({ page = 1, page_size = 20, course_id, status = "", search = "" }) => ({
        url: "/batches",
        params: { page, page_size, course_id, status, search },
      }),
      transformResponse: (r: Envelope<Paginated<Batch>>) => r.data as Paginated<Batch>,
      providesTags: ["BatchList"],
    }),

    getBatch: build.query<Batch, number>({
      query: (id) => `/batches/${id}`,
      transformResponse: (r: Envelope<Batch>) => r.data as Batch,
      providesTags: (_r, _e, id) => [{ type: "Batch", id }],
    }),

    createBatch: build.mutation<Batch, CreateBatchInput>({
      query: (body) => ({ url: "/batches", method: "POST", body }),
      transformResponse: (r: Envelope<Batch>) => r.data as Batch,
      invalidatesTags: ["BatchList"],
    }),

    updateBatch: build.mutation<Batch, { id: number; body: UpdateBatchInput }>({
      query: ({ id, body }) => ({ url: `/batches/${id}`, method: "PUT", body }),
      transformResponse: (r: Envelope<Batch>) => r.data as Batch,
      invalidatesTags: (_r, _e, { id }) => [{ type: "Batch", id }, "BatchList"],
    }),

    deleteBatch: build.mutation<void, number>({
      query: (id) => ({ url: `/batches/${id}`, method: "DELETE" }),
      invalidatesTags: ["BatchList"],
    }),

    setBatchPublished: build.mutation<Batch, { id: number; published: boolean }>({
      query: ({ id, published }) => ({
        url: `/batches/${id}/${published ? "publish" : "unpublish"}`,
        method: "POST",
      }),
      transformResponse: (r: Envelope<Batch>) => r.data as Batch,
      invalidatesTags: (_r, _e, { id }) => [{ type: "Batch", id }, "BatchList"],
    }),

    // --- Module <-> tutor assignment ---
    assignTutor: build.mutation<
      void,
      { batchId: number; moduleId: number; body: AssignTutorInput }
    >({
      query: ({ batchId, moduleId, body }) => ({
        url: `/batches/${batchId}/modules/${moduleId}/assign`,
        method: "PUT",
        body,
      }),
      invalidatesTags: (_r, _e, { batchId }) => [
        { type: "Batch", id: batchId },
        "BatchTutors",
      ],
    }),
    unassignTutor: build.mutation<void, { batchId: number; moduleId: number }>({
      query: ({ batchId, moduleId }) => ({
        url: `/batches/${batchId}/modules/${moduleId}/assign`,
        method: "DELETE",
      }),
      invalidatesTags: (_r, _e, { batchId }) => [
        { type: "Batch", id: batchId },
        "BatchTutors",
      ],
    }),
    listBatchTutors: build.query<TutorSummary[], number>({
      query: (batchId) => `/batches/${batchId}/tutors`,
      transformResponse: (r: Envelope<TutorSummary[]>) => r.data ?? [],
      providesTags: ["BatchTutors"],
    }),

    listBatchStudents: build.query<
      Paginated<StudentSummary>,
      { batchId: number; page?: number; page_size?: number }
    >({
      query: ({ batchId, page = 1, page_size = 20 }) => ({
        url: `/batches/${batchId}/students`,
        params: { page, page_size },
      }),
      transformResponse: (r: Envelope<Paginated<StudentSummary>>) =>
        r.data as Paginated<StudentSummary>,
      providesTags: ["BatchStudents"],
    }),
    enrollStudents: build.mutation<EnrollResult, { batchId: number; studentIds: number[] }>({
      query: ({ batchId, studentIds }) => ({
        url: `/batches/${batchId}/students/enroll`,
        method: "POST",
        body: { student_ids: studentIds },
      }),
      transformResponse: (r: Envelope<EnrollResult>) => r.data as EnrollResult,
      invalidatesTags: (_r, _e, { batchId }) => [
        { type: "Batch", id: batchId },
        "BatchStudents",
      ],
    }),
    removeBatchStudent: build.mutation<void, { batchId: number; studentId: number }>({
      query: ({ batchId, studentId }) => ({
        url: `/batches/${batchId}/students/${studentId}`,
        method: "DELETE",
      }),
      invalidatesTags: (_r, _e, { batchId }) => [
        { type: "Batch", id: batchId },
        "BatchStudents",
      ],
    }),
    importBatchStudents: build.mutation<ImportResult, { batchId: number; file: File }>({
      query: ({ batchId, file }) => {
        const form = new FormData();
        form.append("file", file);
        return { url: `/batches/${batchId}/students/import`, method: "POST", body: form };
      },
      transformResponse: (r: Envelope<ImportResult>) => r.data as ImportResult,
      invalidatesTags: (_r, _e, { batchId }) => [
        { type: "Batch", id: batchId },
        "BatchStudents",
      ],
    }),

    listDrive: build.query<DriveNode[], { batchId: number; parentId?: number }>({
      query: ({ batchId, parentId }) => ({
        url: `/batches/${batchId}/drive`,
        params: parentId ? { parent_id: parentId } : {},
      }),
      transformResponse: (r: Envelope<DriveNode[]>) => r.data ?? [],
      providesTags: ["Drive"],
    }),
    createFolder: build.mutation<
      DriveNode,
      { batchId: number; name: string; parentId?: number }
    >({
      query: ({ batchId, name, parentId }) => ({
        url: `/batches/${batchId}/drive/folders`,
        method: "POST",
        body: { name, parent_id: parentId ?? null },
      }),
      transformResponse: (r: Envelope<DriveNode>) => r.data as DriveNode,
      invalidatesTags: ["Drive"],
    }),
    uploadDriveFile: build.mutation<
      DriveNode,
      { batchId: number; file: File; parentId?: number }
    >({
      query: ({ batchId, file, parentId }) => {
        const form = new FormData();
        form.append("file", file);
        if (parentId) form.append("parent_id", String(parentId));
        return { url: `/batches/${batchId}/drive/files`, method: "POST", body: form };
      },
      transformResponse: (r: Envelope<DriveNode>) => r.data as DriveNode,
      invalidatesTags: ["Drive"],
    }),
    renameNode: build.mutation<DriveNode, { batchId: number; nodeId: number; name: string }>({
      query: ({ batchId, nodeId, name }) => ({
        url: `/batches/${batchId}/drive/${nodeId}`,
        method: "PUT",
        body: { name },
      }),
      transformResponse: (r: Envelope<DriveNode>) => r.data as DriveNode,
      invalidatesTags: ["Drive"],
    }),
    deleteNode: build.mutation<void, { batchId: number; nodeId: number }>({
      query: ({ batchId, nodeId }) => ({
        url: `/batches/${batchId}/drive/${nodeId}`,
        method: "DELETE",
      }),
      invalidatesTags: ["Drive"],
    }),
  }),
});

export const {
  useListBatchesQuery,
  useGetBatchQuery,
  useCreateBatchMutation,
  useUpdateBatchMutation,
  useDeleteBatchMutation,
  useSetBatchPublishedMutation,
  useAssignTutorMutation,
  useUnassignTutorMutation,
  useListBatchTutorsQuery,
  useListBatchStudentsQuery,
  useEnrollStudentsMutation,
  useRemoveBatchStudentMutation,
  useImportBatchStudentsMutation,
  useListDriveQuery,
  useCreateFolderMutation,
  useUploadDriveFileMutation,
  useRenameNodeMutation,
  useDeleteNodeMutation,
} = batchesApi;
