// Mirrors the Go API's httpx.Envelope and DTOs.

export interface Envelope<T> {
  success: boolean;
  message?: string;
  data?: T;
  error?: string;
}

export interface Paginated<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface UserView {
  id: number;
  customer_id: number;
  email: string;
  role: string;
  privileges?: string[];
  created_at: string;
  first_name?: string;
  last_name?: string;
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


export type CourseStatus = "draft" | "published";

export interface CourseResource {
  id: number;
  lesson_id?: number;
  name: string;
  url: string;
  content_type: string;
  size_bytes: number;
  created_at: string;
}

export interface CourseLesson {
  id: number;
  title: string;
  content_md: string;
  position: number;
}

export interface CourseModule {
  id: number;
  title: string;
  position: number;
  lessons: CourseLesson[];
}

export interface Course {
  id: number;
  title: string;
  slug: string;
  summary: string;
  description_md: string;
  thumbnail_url?: string;
  status: CourseStatus;
  published_at?: string;
  created_at: string;
  updated_at: string;
  modules?: CourseModule[];
}

export interface CreateCourseInput {
  title: string;
  summary?: string;
  description_md?: string;
}

export interface UpdateCourseInput {
  title: string;
  summary: string;
  description_md: string;
}

export interface ModuleInput {
  title: string;
  position?: number;
}

export interface LessonInput {
  title: string;
  content_md: string;
  position?: number;
}

export interface Address {
  id: number;
  local_address: string;
  city: string;
  state: string;
  country: string;
}

export interface AddressInput {
  local_address: string;
  city: string;
  state: string;
  country: string;
}

export interface Tutor {
  id: number;
  first_name: string;
  last_name: string;
  email: string;
  phone_no: string;
  designation: string;
  profile_image_url?: string;
  address?: Address;
  created_at: string;
  updated_at: string;
}

export interface CreateTutorInput {
  first_name: string;
  last_name: string;
  email: string;
  phone_no: string;
  designation: string;
  address?: AddressInput;
}

export type UpdateTutorInput = CreateTutorInput;

export interface Student {
  id: number;
  first_name: string;
  last_name: string;
  email: string;
  phone_no: string;
  profile_image_url?: string;
  address?: Address;
  created_at: string;
  updated_at: string;
}

export interface CreateStudentInput {
  first_name: string;
  last_name: string;
  email: string;
  phone_no: string;
  address?: AddressInput;
}

export type UpdateStudentInput = CreateStudentInput;

export type BatchStatus = "draft" | "published";

export interface TutorSummary {
  id: number;
  first_name: string;
  last_name: string;
  email: string;
}

export interface StudentSummary {
  id: number;
  first_name: string;
  last_name: string;
  email: string;
  phone_no: string;
}

export interface ModuleAssignment {
  course_module_id: number;
  module_title: string;
  module_position: number;
  tutor?: TutorSummary;
  start_date?: string;
  expected_end_date?: string;
}

export interface Batch {
  id: number;
  course_id: number;
  name: string;
  status: BatchStatus;
  published_at?: string;
  created_at: string;
  updated_at: string;
  modules?: ModuleAssignment[];
  tutor_count: number;
  student_count: number;
}

export interface CreateBatchInput {
  course_id: number;
  name: string;
}

export interface UpdateBatchInput {
  name: string;
}

export interface AssignTutorInput {
  tutor_id: number;
  start_date: string;
  expected_end_date: string;
}

export interface ImportResult {
  imported: number;
  skipped: { row: number; reason: string }[];
}

export type DriveNodeType = "folder" | "file";

export interface DriveNode {
  id: number;
  parent_id?: number;
  name: string;
  type: DriveNodeType;
  url?: string;
  content_type?: string;
  size_bytes?: number;
  created_at: string;
}
