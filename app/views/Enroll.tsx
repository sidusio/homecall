import { StyleSheet, Text, View, Pressable } from 'react-native';
import { CameraView, useCameraPermissions, useMicrophonePermissions } from 'expo-camera/next';
import {EnrollmentData, isEnrollmentData} from "../services/enrollment";

const homecallProtocolPrefix = 'homecall://';

export default function Enroll(props: {
  onEnroll: (data: EnrollmentData) => void,
}){
  const { onEnroll } = props;
  const [permission, requestPermission] = useCameraPermissions();
  const [microphonePermission, requestMicrophonePermission] = useMicrophonePermissions();

  if (!permission || !microphonePermission) {
    return <View/>;
  }

  if (!permission.granted || !microphonePermission.granted) {
    // Camera permissions are not granted yet
    return (
      <View style={styles.container}>
        <Text style={styles.heading}>
          Välkommen till HomeCall
        </Text>

        <Text style={styles.text}>
          För att kunna registrera din enhet behöver vi ha tillgång till enhetens kamera och mikrofon.
        </Text>

        <View style={styles.buttons}>
          <Pressable
              style={[
                styles.button,
                permission.granted ? styles.permissionGranted : styles.permissionAsked
              ]}
              onPress={requestPermission}
            >
              { permission.granted ? <Text style={styles.buttonText}>Kamera tillåten</Text> : <Text style={styles.buttonText}>Tillåt Kamera</Text> }
            </Pressable>

          <Pressable
            style={[
              styles.button,
              microphonePermission.granted ? styles.permissionGranted : styles.permissionAsked,
            ]}
            onPress={requestMicrophonePermission}
          >
            { microphonePermission.granted ? <Text style={styles.buttonText}>Mikrofon tillåten</Text> : <Text style={styles.buttonText}>Tillåt Mikrofon</Text> }
          </Pressable>
        </View>

        <Text style={styles.remember}>
          Det är också viktigt att du startar ett samtal när du registrerat enheten för att ge samtalsmodulen tillgång till mikrofon och kamera.
        </Text>
      </View>
    );
  }

  const barcodeScanned = ({data}: {data: string}) => {
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
    <View style={styles.cameraContainer}>
      <Text style={styles.information}>
        Skanna QR-koden för att registrera din enhet.
      </Text>

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
    alignItems: 'center',
    padding: 24,
  },
  cameraContainer: {
    flex: 1,
    justifyContent: 'center',
  },
  camera: {
    flex: 1,
    height: '100%',
    width: '100%',
  },
  buttons: {
    display: 'flex',
    flexDirection: 'row',
    gap: 16,
  },
  button: {
    padding: 10,
    backgroundColor: 'rgb(67, 107, 177)',
    borderRadius: 100,
    paddingLeft: 32,
    paddingRight: 32,
    paddingBottom: 22,
    paddingTop: 22,
  },
  buttonText: {
    color: 'rgb(255, 255, 255)',
  },
  permissionGranted: {
    opacity: 0.5,
  },
  permissionAsked: {
    backgroundColor: 'rgb(67, 107, 177)',
  },
  text: {
    fontSize: 16,
    textAlign: 'center',
    marginBottom: 24,
  },
  heading: {
    fontSize: 24,
    textAlign: 'center',
    marginBottom: 10,
  },
  information: {
    textAlign: 'center',
    backgroundColor: 'rgb(67, 107, 177)',
    color: 'white',
    paddingTop: 60,
    paddingBottom: 10,
    paddingLeft: 20,
    paddingRight: 20,
  },
  load: {
    display: 'flex',
  },
  notLoad: {
    display: 'none',
  },
  remember: {
    backgroundColor: 'rgba(67, 107, 177, 0.1)',
    padding: 16,
    borderRadius: 10,
    fontSize: 13,
    width: '100%',
    textAlign: 'center',
    marginTop: 34,
  }
});
