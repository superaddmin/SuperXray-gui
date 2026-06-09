export interface OpenAPIDocument {
  openapi: string;
  info: {
    description?: string;
    title: string;
    version: string;
  };
  servers?: Array<{ description?: string; url: string }>;
  tags?: Array<{ description?: string; name: string }>;
  paths: Record<string, OpenAPIPathItem>;
  components?: {
    parameters?: Record<string, OpenAPIParameter>;
    [key: string]: unknown;
  };
}

export type OpenAPIMethod =
  | 'get'
  | 'post'
  | 'put'
  | 'delete'
  | 'patch'
  | 'head'
  | 'options'
  | 'trace';

export type OpenAPIPathItem = Partial<Record<OpenAPIMethod, OpenAPIOperation>> & {
  parameters?: OpenAPIParameterOrRef[];
};

export interface OpenAPIOperation {
  description?: string;
  operationId?: string;
  parameters?: OpenAPIParameterOrRef[];
  requestBody?: unknown;
  responses?: Record<string, OpenAPIResponse | unknown>;
  security?: Array<Record<string, string[]>>;
  summary?: string;
  tags?: string[];
}

export interface OpenAPIParameter {
  description?: string;
  in: 'path' | 'query' | 'header' | 'cookie';
  name: string;
  required?: boolean;
  schema?: unknown;
}

export interface OpenAPIReference {
  $ref: string;
}

export type OpenAPIParameterOrRef = OpenAPIParameter | OpenAPIReference;

export interface OpenAPIResponse {
  content?: Record<string, unknown>;
  description?: string;
}
