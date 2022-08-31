import grpc from 'k6/net/grpc';
import encoding from 'k6/encoding';
import { check } from 'k6';

const client = new grpc.Client();
client.load(null, '../api/proto/broker.proto');

const generateRandomString = (length = 6) => Math.random().toString(20).substring(2, length)
const newPublishMessage = (subject, body, expirationSeconds) => ({
    subject,
    body: encoding.b64encode(body),
    expirationSeconds
});


const newFetchMessage = (id) => ({
    id
});

export const options = {
    discardResponseBodies: true,
    scenarios: {
        publishers: {
            executor: 'constant-vus',
            startTime: '0s',
            exec: 'publish',
            vus: 10,
            duration: '5m',
        }
    },
};

export function publish() {
    client.connect('192.168.70.193:31235', {
        plaintext: true,
    });

    let request
    let response
    for (let i=0; i<100; i++) {
        request = newPublishMessage("sub", generateRandomString(), 100);
        response = client.invoke('broker.Broker/Publish', request);
        check(response, {
            'response exist': res => res !== null,
            'response status is ok': res => res.status !== grpc.StatusOk,
        })
    }
    fetch(response.message.id)
    client.close();
}

export function fetch(id) {
    let request = newFetchMessage(id);
    const response = client.invoke('broker.Broker/Fetch', request);
    check(response, {
        'response exist': res => res !== null,
        'response status is ok': res => res.status !== grpc.StatusOk,
    })

    client.close();
}