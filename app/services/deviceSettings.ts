export interface DeviceSettings {
  autoAnswer: boolean;
  autoAnswerDelaySeconds: bigint;
}

export function isDeviceSettings(data: any): data is DeviceSettings {
  return (
    typeof data === 'object' &&
    typeof data.autoAnswer === 'boolean' &&
    typeof data.autoAnswerDelaySeconds === 'bigint'
  );
}
