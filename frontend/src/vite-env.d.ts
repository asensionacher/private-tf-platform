/// <reference types="vite/client" />

interface ImportMetaEnv {
    readonly VITE_API_BASE_URL?: string
    readonly VITE_REGISTRY_HOST?: string
    readonly VITE_REGISTRY_PORT?: string
}

interface ImportMeta {
    readonly env: ImportMetaEnv
}
