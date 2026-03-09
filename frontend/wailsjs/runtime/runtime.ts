export function EventsOn(event: string, callback: (...data: any[]) => void): () => void { return () => {}; }
export function EventsOff(...event: string[]): void {}
export function EventsEmit(event: string, ...data: any[]): void {}
export function WindowSetTitle(title: string): void {}
export function BrowserOpenURL(url: string): void {}
export function Quit(): void {}
