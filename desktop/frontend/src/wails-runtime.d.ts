declare module '@wailsio/runtime' {
  export interface CancellablePromise<T> extends Promise<T> {
    cancel(): void
  }

  export const Call: {
    ByID: (methodID: number, ...args: any[]) => CancellablePromise<any>
    ByName: (methodName: string, ...args: any[]) => CancellablePromise<any>
  }

  export const Create: {
    Events: Record<string, (...args: any[]) => any>
  }

  export class WailsEvent<T = unknown> {
    name: string
    data: T
    sender?: string
  }

  export type WailsEventCallback<T = unknown> = (ev: WailsEvent<T>) => void

  export const Events: {
    On: <T = unknown>(eventName: string, callback: WailsEventCallback<T>) => () => void
    Once: <T = unknown>(eventName: string, callback: WailsEventCallback<T>) => () => void
    Off: (...eventNames: string[]) => void
    OffAll: () => void
    Emit: <T = unknown>(name: string, data?: T) => Promise<boolean>
  }
}
