import cv2
import socket
import struct
import time
import argparse
import uuid
import re

# Function to send large messages in chunks
def send_large_message(sock, message, address, chunk_size=60000):
    total_size = len(message)
    num_chunks = (total_size + chunk_size - 1) // chunk_size
    identifier = int(time.time() * 1000) % 4294967296  # Unique identifier based on current time in ms

    # Send chunks
    for i in range(num_chunks):
        chunk = message[i * chunk_size:(i + 1) * chunk_size]
        header = struct.pack('IHH', identifier, i, num_chunks)
        sock.sendto(header + chunk, address)

# Function to parse command-line arguments
def parse_arguments():
    parser = argparse.ArgumentParser(description="Video stream sender configuration")
    parser.add_argument('--ip', type=str, default='224.0.0.1', help='Multicast group IP address')
    parser.add_argument('--port', type=int, default=6001, help='Port number of the receiver')
    parser.add_argument('--device', type=int, default=0, help='video device number')
    return parser.parse_args()


def mac_to_uuid():
    mac = uuid.getnode()
    mac_address = ':'.join(('%012X' % mac)[i:i + 2] for i in range(0, 12, 2))

    # Remove colons or hyphens from MAC address
    clean_mac = re.sub(r'[^a-fA-F0-9]', '', mac_address)

    # Convert hexadecimal MAC address to a decimal integer
    mac_int = int(clean_mac, 16)

    # Format as UUID (v5 uses SHA-1 hashing)
    namespace = uuid.UUID('00000000-0000-0000-0000-000000000000')
    mac_uuid = uuid.uuid5(namespace, str(mac_int))

    return mac_uuid


def main():
    # Parse command-line arguments
    args = parse_arguments()
    receiver_address = (args.ip, args.port)

    # Setup camera
    print("setting up camera")
    camera = cv2.VideoCapture(args.device, cv2.CAP_V4L2)
    camera.set(cv2.CAP_PROP_FOURCC, cv2.VideoWriter_fourcc('M','J','P','G'))
    camera.set(cv2.CAP_PROP_FRAME_WIDTH, 640)
    camera.set(cv2.CAP_PROP_FRAME_HEIGHT, 480)
    camera.set(cv2.CAP_PROP_FPS, 60)

    # Setup UDP socket
    udp_socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM, socket.IPPROTO_UDP)
    udp_socket.setsockopt(socket.IPPROTO_IP, socket.IP_MULTICAST_TTL, 2)  # Multicast TTL

    mac_uuid = mac_to_uuid()
    print(mac_uuid)

    # Frame capture and send
    frame_num = 0
    try:
        while True:
            print("Reading frame...")
            ret, frame = camera.read()
            if not ret:
                break
            #print("checking divider")
            #should we process this frame?
            divider = 4
            frame_num += 1
            if frame_num % divider != 0:
                continue

            # Encode frame as JPEG
            _, buffer = cv2.imencode('.jpg', frame)

            # Get timestamp
            timestamp = time.time()
            #with open ('img_ts.csv', 'a') as file:
            #    file.write(f'{timestamp}\n')

            # Pack timestamp, UUID, and frame together
            message = struct.pack('d16s', timestamp, mac_uuid.bytes) + buffer.tobytes()

            # Send data
            print(f"Sending frame to {receiver_address}...")
            send_large_message(udp_socket, message, receiver_address)
            print(f"Frame sent with UUID: {mac_uuid}")
    finally:
        camera.release()
        udp_socket.close()

if __name__ == "__main__":
    main()
