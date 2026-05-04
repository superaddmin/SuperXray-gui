const encodePath = (value: string | number) => encodeURIComponent(String(value));

export const legacyEndpoints = {
  auth: {
    login: 'login',
    twoFactorEnabled: 'getTwoFactorEnable',
  },
  server: {
    status: 'panel/api/server/status',
    xrayVersions: 'panel/api/server/getXrayVersion',
    restartXray: 'panel/api/server/restartXrayService',
    stopXray: 'panel/api/server/stopXrayService',
    installXray: (version: string) => `panel/api/server/installXray/${encodePath(version)}`,
    updateGeofile: (fileName?: string) =>
      fileName
        ? `panel/api/server/updateGeofile/${encodePath(fileName)}`
        : 'panel/api/server/updateGeofile',
    logs: (count: number) => `panel/api/server/logs/${encodePath(count)}`,
    xrayLogs: (count: number) => `panel/api/server/xraylogs/${encodePath(count)}`,
    database: 'panel/api/server/getDb',
    importDatabase: 'panel/api/server/importDB',
  },
  customGeo: {
    list: 'panel/api/custom-geo/list',
    add: 'panel/api/custom-geo/add',
    update: (id: number) => `panel/api/custom-geo/update/${encodePath(id)}`,
    delete: (id: number) => `panel/api/custom-geo/delete/${encodePath(id)}`,
    download: (id: number) => `panel/api/custom-geo/download/${encodePath(id)}`,
    updateAll: 'panel/api/custom-geo/update-all',
  },
  inbounds: {
    list: 'panel/api/inbounds/list',
    add: 'panel/api/inbounds/add',
    update: (id: number) => `panel/api/inbounds/update/${encodePath(id)}`,
    delete: (id: number) => `panel/api/inbounds/del/${encodePath(id)}`,
    import: 'panel/api/inbounds/import',
    addClient: 'panel/api/inbounds/addClient',
    updateClient: (clientId: string) => `panel/api/inbounds/updateClient/${encodePath(clientId)}`,
    deleteClient: (id: number, clientId: string) =>
      `panel/api/inbounds/${encodePath(id)}/delClient/${encodePath(clientId)}`,
    resetClientTraffic: (id: number, email: string) =>
      `panel/api/inbounds/${encodePath(id)}/resetClientTraffic/${encodePath(email)}`,
    resetAllTraffics: 'panel/api/inbounds/resetAllTraffics',
    resetAllClientTraffics: (id: number) =>
      `panel/api/inbounds/resetAllClientTraffics/${encodePath(id)}`,
    deleteDepletedClients: (id: number) =>
      `panel/api/inbounds/delDepletedClients/${encodePath(id)}`,
    onlines: 'panel/api/inbounds/onlines',
    lastOnline: 'panel/api/inbounds/lastOnline',
    clientIps: (email: string) => `panel/api/inbounds/clientIps/${encodePath(email)}`,
    clearClientIps: (email: string) => `panel/api/inbounds/clearClientIps/${encodePath(email)}`,
  },
  xray: {
    setting: 'panel/xray/',
    update: 'panel/xray/update',
    result: 'panel/xray/getXrayResult',
    outboundsTraffic: 'panel/xray/getOutboundsTraffic',
    resetOutboundsTraffic: 'panel/xray/resetOutboundsTraffic',
    testOutbound: 'panel/xray/testOutbound',
    warp: (action: string) => `panel/xray/warp/${encodePath(action)}`,
    nord: (action: string) => `panel/xray/nord/${encodePath(action)}`,
  },
  settings: {
    all: 'panel/setting/all',
    defaultSettings: 'panel/setting/defaultSettings',
    update: 'panel/setting/update',
    updateUser: 'panel/setting/updateUser',
    restartPanel: 'panel/setting/restartPanel',
  },
} as const;
