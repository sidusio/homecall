// @generated by protoc-gen-es v1.8.0
// @generated from file homecall/v1alpha/device_service.proto (package homecall.v1alpha, syntax proto3)
/* eslint-disable */
// @ts-nocheck

import type { BinaryReadOptions, FieldList, JsonReadOptions, JsonValue, PartialMessage, PlainMessage } from "@bufbuild/protobuf";
import { Message, proto3 } from "@bufbuild/protobuf";
import type { DeviceSettings } from "./settings_pb.js";

/**
 * EnrollRequest is the request to enroll a device.
 *
 * @generated from message homecall.v1alpha.EnrollRequest
 */
export declare class EnrollRequest extends Message<EnrollRequest> {
  /**
   * The enrollment key is a secret key that is used to enroll a device.
   * This key is generated by the service and shared with the device via the office app.
   *
   * @generated from field: string enrollment_key = 1;
   */
  enrollmentKey: string;

  /**
   * The public key is the public key of the device that is used to encrypt the call information.
   * This key is generated by the device and shared with the service during enrollment.
   * The public key is a PEM encoded RSA public key.
   *
   * @generated from field: string public_key = 2;
   */
  publicKey: string;

  constructor(data?: PartialMessage<EnrollRequest>);

  static readonly runtime: typeof proto3;
  static readonly typeName = "homecall.v1alpha.EnrollRequest";
  static readonly fields: FieldList;

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): EnrollRequest;

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): EnrollRequest;

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): EnrollRequest;

  static equals(a: EnrollRequest | PlainMessage<EnrollRequest> | undefined, b: EnrollRequest | PlainMessage<EnrollRequest> | undefined): boolean;
}

/**
 * EnrollResponse is the response to enrolling a device.
 *
 * @generated from message homecall.v1alpha.EnrollResponse
 */
export declare class EnrollResponse extends Message<EnrollResponse> {
  /**
   * The device ID is the unique identifier for the device.
   *
   * @generated from field: string device_id = 1;
   */
  deviceId: string;

  /**
   * The device settings are the settings for the device.
   * These settings are used to configure the device for the user.
   *
   * @generated from field: homecall.v1alpha.DeviceSettings settings = 2;
   */
  settings?: DeviceSettings;

  /**
   * The name of the device.
   *
   * @generated from field: string name = 3;
   */
  name: string;

  constructor(data?: PartialMessage<EnrollResponse>);

  static readonly runtime: typeof proto3;
  static readonly typeName = "homecall.v1alpha.EnrollResponse";
  static readonly fields: FieldList;

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): EnrollResponse;

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): EnrollResponse;

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): EnrollResponse;

  static equals(a: EnrollResponse | PlainMessage<EnrollResponse> | undefined, b: EnrollResponse | PlainMessage<EnrollResponse> | undefined): boolean;
}

/**
 * GetCallDetailsRequest is the request to join a call.
 * Token is passed in the Authorization header as a bearer token.
 *
 * @generated from message homecall.v1alpha.GetCallDetailsRequest
 */
export declare class GetCallDetailsRequest extends Message<GetCallDetailsRequest> {
  /**
   * The call ID is the unique identifier for the call.
   *
   * @generated from field: string call_id = 1;
   */
  callId: string;

  constructor(data?: PartialMessage<GetCallDetailsRequest>);

  static readonly runtime: typeof proto3;
  static readonly typeName = "homecall.v1alpha.GetCallDetailsRequest";
  static readonly fields: FieldList;

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): GetCallDetailsRequest;

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): GetCallDetailsRequest;

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): GetCallDetailsRequest;

  static equals(a: GetCallDetailsRequest | PlainMessage<GetCallDetailsRequest> | undefined, b: GetCallDetailsRequest | PlainMessage<GetCallDetailsRequest> | undefined): boolean;
}

/**
 * GetCallDetailsResponse is the response to join a call.
 *
 * @generated from message homecall.v1alpha.GetCallDetailsResponse
 */
export declare class GetCallDetailsResponse extends Message<GetCallDetailsResponse> {
  /**
   * The room ID is the unique identifier for the jitsi room.
   * Used to join the call.
   *
   * @generated from field: string jitsi_room_id = 1;
   */
  jitsiRoomId: string;

  /**
   * The jitsi jwt is the jwt token used to authenticate the device with jitsi.
   *
   * @generated from field: string jitsi_jwt = 2;
   */
  jitsiJwt: string;

  /**
   * The call ID is the unique identifier for the call.
   *
   * @generated from field: string call_id = 3;
   */
  callId: string;

  constructor(data?: PartialMessage<GetCallDetailsResponse>);

  static readonly runtime: typeof proto3;
  static readonly typeName = "homecall.v1alpha.GetCallDetailsResponse";
  static readonly fields: FieldList;

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): GetCallDetailsResponse;

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): GetCallDetailsResponse;

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): GetCallDetailsResponse;

  static equals(a: GetCallDetailsResponse | PlainMessage<GetCallDetailsResponse> | undefined, b: GetCallDetailsResponse | PlainMessage<GetCallDetailsResponse> | undefined): boolean;
}

/**
 * UpdateNotificationTokenRequest is the request to update the FCM token.
 *
 * @generated from message homecall.v1alpha.UpdateNotificationTokenRequest
 */
export declare class UpdateNotificationTokenRequest extends Message<UpdateNotificationTokenRequest> {
  /**
   * The notification_token is the token used to send push notifications to the device.
   *
   * @generated from field: string notification_token = 1;
   */
  notificationToken: string;

  constructor(data?: PartialMessage<UpdateNotificationTokenRequest>);

  static readonly runtime: typeof proto3;
  static readonly typeName = "homecall.v1alpha.UpdateNotificationTokenRequest";
  static readonly fields: FieldList;

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): UpdateNotificationTokenRequest;

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): UpdateNotificationTokenRequest;

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): UpdateNotificationTokenRequest;

  static equals(a: UpdateNotificationTokenRequest | PlainMessage<UpdateNotificationTokenRequest> | undefined, b: UpdateNotificationTokenRequest | PlainMessage<UpdateNotificationTokenRequest> | undefined): boolean;
}

/**
 * UpdateNotificationTokenResponse is the response to updating the FCM token.
 *
 * @generated from message homecall.v1alpha.UpdateNotificationTokenResponse
 */
export declare class UpdateNotificationTokenResponse extends Message<UpdateNotificationTokenResponse> {
  constructor(data?: PartialMessage<UpdateNotificationTokenResponse>);

  static readonly runtime: typeof proto3;
  static readonly typeName = "homecall.v1alpha.UpdateNotificationTokenResponse";
  static readonly fields: FieldList;

  static fromBinary(bytes: Uint8Array, options?: Partial<BinaryReadOptions>): UpdateNotificationTokenResponse;

  static fromJson(jsonValue: JsonValue, options?: Partial<JsonReadOptions>): UpdateNotificationTokenResponse;

  static fromJsonString(jsonString: string, options?: Partial<JsonReadOptions>): UpdateNotificationTokenResponse;

  static equals(a: UpdateNotificationTokenResponse | PlainMessage<UpdateNotificationTokenResponse> | undefined, b: UpdateNotificationTokenResponse | PlainMessage<UpdateNotificationTokenResponse> | undefined): boolean;
}