# locust_tests

This folder contains locust tests for the project.

## message_generator.py

This is a locust file that can be used to send test messages to the Service Bus Topic for the subsciber-app to process.

To run, copy `sample.env` under `locust_tests` to `.env` and fill out the values in the message_generator.py section.

Then run the following command from the `locust_tests` directory:

```bash
# Run message_generator.py with 6 users
locust -f message_generator.py  --autostart --spawn-rate 10 --users 6
```

You can use the `w` and `s` keys to increase or decrease the number of users by 1.
You can use `W` and `S` (note the capitalisation) to increase or decrease the number of users by 10.


## openai_load.py

This locust file generates load against the OpenAI API.
The purpose is to consume tokens and requests from the rate-limit for the endpoint that the subscriber-app uses to call the OpenAI API.

To run, copy `sample.env` under `locust_tests` to `.env` and fill out the values in the openai_load.py section.

Then run the following command from the `locust_tests` directory:

```bash
# Run openai_load.py with 6 users
locust -f openai_load.py --web-port 8091 --autostart --spawn-rate 10 --users 6
```