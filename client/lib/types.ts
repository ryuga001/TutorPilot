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

/** Which kind of principal an attendance row belongs to. */
export type SubjectType = "admin" | "tutor" | "student";

export interface UserView {
  id: number;
  customer_id: number;
  email: string;
  first_name: string;
  last_name: string;
  role: string;
  privileges?: string[];
  created_at: string;
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

export interface ChangePasswordInput {
  current_password: string;
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

export type LectureStatus = "scheduled" | "live" | "ended" | "cancelled";

/** A recording is not ready when a lecture ends: egress finalises asynchronously,
 *  so it passes through "processing" before the file exists. */
export type RecordingStatus =
  | "none"
  | "starting"
  | "recording"
  | "processing"
  | "ready"
  | "failed";

export interface Lecture {
  id: number;
  batch_id: number;
  module_id?: number;
  tutor_id?: number;
  title: string;
  description: string;
  status: LectureStatus;
  room_name?: string;

  recording_enabled: boolean;
  recording_status: RecordingStatus;
  recording_url?: string;
  recording_duration_seconds?: number;
  recording_size_bytes?: number;

  /** Scheduled start. */
  start_time: string;
  /** When the lecture was actually started. */
  actual_start_at?: string;
  end_time?: string;

  created_at: string;
  updated_at: string;

  /** Denormalised by the API so a listing needs no follow-up requests. */
  batch_name?: string;
  course_title?: string;
  module_title?: string;
  tutor_name?: string;

  /** Whether the current user may publish audio/video in this lecture. */
  can_publish: boolean;
}

export interface CreateLectureInput {
  batch_id: number;
  module_id?: number;
  tutor_id?: number;
  title: string;
  description?: string;
  recording_enabled?: boolean;
  start_time: string;
  end_time?: string;
}

export type UpdateLectureInput = Omit<CreateLectureInput, "batch_id">;

export interface LectureJoinResponse {
  token: string;
  room_name: string;
  identity: string;
  can_publish: boolean;
}

export interface LectureAttendance {
  user_id: number;
  subject_type: SubjectType;
  subject_id?: number;
  display_name: string;
  joined_at: string;
  left_at?: string;
  seconds_present?: number;
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
  /** Only present in the response to creating the tutor: the login is created
   *  atomically with the record, and this is the one time its password is
   *  readable. */
  temp_password?: string;
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
  /** Only present in the response to creating the student: the login is created
   *  atomically with the record, and this is the one time its password is
   *  readable. */
  temp_password?: string;
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

export interface EnrollResult {
  enrolled: number;
  not_found?: number[];
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
  /** Managed by the application — the lecture recordings folder. Cannot be
   *  renamed or deleted. */
  is_system?: boolean;
  created_at: string;
}
