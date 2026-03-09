function makeClass() {
  return class {
    constructor(data?: any) {
      if (data) Object.assign(this, data);
    }
  };
}

export namespace main {
  export class SpotifyMetadataRequest extends makeClass() {}
  export class DownloadRequest extends makeClass() {}
  export class DownloadResponse extends makeClass() {}
  export class LyricsDownloadRequest extends makeClass() {}
  export class LyricsDownloadResponse extends makeClass() {}
  export class CoverDownloadRequest extends makeClass() {}
  export class CoverDownloadResponse extends makeClass() {}
  export class HeaderDownloadRequest extends makeClass() {}
  export class HeaderDownloadResponse extends makeClass() {}
  export class GalleryImageDownloadRequest extends makeClass() {}
  export class GalleryImageDownloadResponse extends makeClass() {}
  export class AvatarDownloadRequest extends makeClass() {}
  export class AvatarDownloadResponse extends makeClass() {}
  export class SpotifySearchRequest extends makeClass() {}
  export class CheckFileExistenceRequest extends makeClass() {}
  export class CheckFileExistenceResult extends makeClass() {}
}

export namespace backend {
  export class DownloadQueueInfo {
    is_downloading: boolean = false;
    queue: any[] = [];
    current_speed: number = 0;
    total_downloaded: number = 0;
    session_start_time: number = 0;
    queued_count: number = 0;
    completed_count: number = 0;
    failed_count: number = 0;
    skipped_count: number = 0;
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }

  export class ProgressInfo extends makeClass() {}
  export class DownloadItem extends makeClass() {}
  export class HistoryItem extends makeClass() {}
  export class FetchHistoryItem extends makeClass() {}

  export class FileInfo {
    name: string = "";
    path: string = "";
    size: number = 0;
    is_dir: boolean = false;
    extension: string = "";
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }

  // Toutes les propriétés utilisées par FileManagerPage
  export class AudioMetadata {
    title: string = "";
    artist: string = "";
    album: string = "";
    album_artist: string = "";
    track_number: number = 0;
    disc_number: number = 0;
    year: string = "";
    duration: number = 0;
    bit_rate: number = 0;
    sample_rate: number = 0;
    channels: number = 0;
    format: string = "";
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }

  // Toutes les propriétés utilisées par SearchBar
  export class SearchResult {
    id: string = "";
    name: string = "";
    artists: string = "";
    album: string = "";
    type: string = "";
    // image seul ET images (tableau) — le composant utilise les deux
    image: string = "";
    images: string = "";
    external_urls: string = "";
    is_explicit: boolean = false;
    duration_ms: number = 0;
    release_date: string = "";
    owner: string = "";
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }

  export class SearchResponse {
    tracks: SearchResult[] = [];
    albums: SearchResult[] = [];
    artists: SearchResult[] = [];
    playlists: SearchResult[] = [];
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }

  export class RenamePreview {
    old_name: string = "";
    new_name: string = "";
    error: string = "";
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }

  export class RenameResult {
    success: boolean = false;
    old_path: string = "";
    new_path: string = "";
    error: string = "";
    constructor(data?: any) { if (data) Object.assign(this, data); }
  }

  export class AnalysisResult extends makeClass() {}
  export class ConvertAudioResult extends makeClass() {}
}
