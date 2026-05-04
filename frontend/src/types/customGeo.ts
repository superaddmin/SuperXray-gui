export type CustomGeoType = 'geoip' | 'geosite';

export interface CustomGeoResource {
  alias: string;
  createdAt: number;
  id: number;
  lastModified: string;
  lastUpdatedAt: number;
  localPath: string;
  type: CustomGeoType;
  updatedAt: number;
  url: string;
}

export interface CustomGeoForm {
  alias: string;
  type: CustomGeoType;
  url: string;
}
