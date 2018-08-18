# Worker
## Subscription Partition
This work is triggered when there is any document change. It's main purpose is to allow the notification stream to be parallel processed. Let's imagine a scenario where a given resource is very popular and it can have thousands of subscribers.

Behind de scenes, when a subscription is created it is associated in a resource partition. The partition has a fixed value defined at the configuration file. When the partition is full, another one is created. There is no limit of partitions quantity.

The actual job is to trigger *Subscription Spread* for each partition.

## Subscription Spread
After the message is received, this worker knows the partition that should be processed. Then, it goes into the database to fetch all the subscriptions inside that partition. For each subscription a trigger is generated to *Subscription Delivery*.

## Subscription Delivery
Here is the final of the stream and where all the action happens.