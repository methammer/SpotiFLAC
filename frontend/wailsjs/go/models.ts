export namespace main {
  export class SpotifyMetadataRequest {
    url: string = ""; batch: boolean = true; delay: number = 1.0; timeout: number = 300.0;
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }
  export class DownloadRequest {
    [key: string]: any;
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }
  export class LyricsDownloadRequest {
    [key: string]: any;
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }
  export class CoverDownloadRequest {
    [key: string]: any;
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }
  export class HeaderDownloadRequest {
    [key: string]: any;
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }
  export class GalleryImageDownloadRequest {
    [key: string]: any;
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }
  export class AvatarDownloadRequest {
    [key: string]: any;
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }
}

export namespace backend {
  export class DownloadQueueInfo {
    [key: string]: any;
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }
  export class RenamePreview {
    [key: string]: any;
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }
  export class RenameResult {
    [key: string]: any;
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }
  export class SearchResponse {
    [key: string]: any;
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }
}
