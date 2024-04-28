import { StatusBar } from 'expo-status-bar';
import {Button, StyleSheet, Text, View} from 'react-native';
import { CameraView, useCameraPermissions } from 'expo-camera/next';

const homecallProtocolPrefix = 'homecall://';

interface EnrollmentData {
  enrollmentId: string;
  enrollmentKey: string;
  enrollmentUrl: string;
}

function isEnrollmentData(data: any): data is EnrollmentData {
  return (
    typeof data === 'object' &&
    typeof data.enrollmentId === 'string' &&
    typeof data.enrollmentKey === 'string' &&
    typeof data.enrollmentUrl === 'string'
  );
}

export default function Enroll(props: {
  onEnroll: (data: EnrollmentData) => void,
}){
  const { onEnroll } = props;
  const [permission, requestPermission] = useCameraPermissions();

  if (!permission) {
    // Camera permissions are still loading
    return <View />;
  }

  if (!permission.granted) {
    // Camera permissions are not granted yet
    return (
      <View style={styles.container}>
        <Text style={{ textAlign: 'center' }}>We need your permission to show the camera</Text>
        <Button onPress={requestPermission} title="grant permission" />
      </View>
    );
  }

  const barcodeScanned = ({data}) => {
    if (!data.startsWith(homecallProtocolPrefix)) {
      console.error('Invalid homecall protocol', data);
    }

    const enrollmentData: unknown = JSON.parse(data.slice(homecallProtocolPrefix.length));
    if (!isEnrollmentData(enrollmentData)) {
      console.error('Invalid enrollment data', enrollmentData);
      return;
    }

    onEnroll(enrollmentData)

  }


  return (
    <View style={styles.container}>
      <CameraView
        style={styles.camera}
        facing={'back'}
        barcodeScannerSettings={{
          barcodeTypes: ['qr'],
        }}
        onBarcodeScanned={barcodeScanned}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
  },
  camera: {
    flex: 1,
  },
  button: {
    flex: 1,
    alignSelf: 'flex-end',
    alignItems: 'center',
  },
  text: {
    fontSize: 24,
    fontWeight: 'bold',
    color: 'white',
  },
});
