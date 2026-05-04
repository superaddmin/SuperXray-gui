type JsonPrimitive = string | number | boolean | null;
export type JsonValue = JsonPrimitive | JsonObject | JsonValue[];

export interface JsonObject {
  [key: string]: JsonValue;
}

type FormPrimitive = string | number | boolean;
export type FormValue = FormPrimitive | FormPrimitive[] | null | undefined;
export type FormRecord = Record<string, FormValue>;

export interface LegacyResponse<T> {
  success: boolean;
  msg: string;
  obj: T;
}

export interface ApiErrorDetails<T = unknown> {
  status?: number;
  endpoint?: string;
  response?: LegacyResponse<T> | T;
  sessionExpired?: boolean;
}
