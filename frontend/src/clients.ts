import { createPromiseClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

// Import service definition that you want to connect to.
import { OfficeService } from "./../gen/connect/homecall/v1alpha/office_service_connect";
import { DeviceService } from "./../gen/connect/homecall/v1alpha/device_service_connect";
import { TenantService } from "./../gen/connect/homecall/v1alpha/tenant_service_connect";

// The transport defines what type of endpoint we're hitting.
// In our example we'll be communicating with a Connect endpoint.
// If your endpoint only supports gRPC-web, make sure to use
// `createGrpcWebTransport` instead.
const transport = createConnectTransport({
  baseUrl: "/api",
});

// Here we make the client itself, combining the service
// definition with the transport.
const officeClient = createPromiseClient(OfficeService, transport);
const deviceClient = createPromiseClient(DeviceService, transport);
const tenantClient = createPromiseClient(TenantService, transport);

export { officeClient, deviceClient, tenantClient };
