import fs from 'fs';
import webpush from 'web-push';
import sshpk from 'sshpk';

import type { PoolConnection } from 'promise-mysql';
import { Notification } from '../proto/xsuportal/resources/notification_pb';

export class Notifier {
  static WEBPUSH_VAPID_PRIVATE_KEY_PATH = '../vapid_private.pem';
  static WEBPUSH_SUBJECT = 'xsuportal@example.com';
  static VAPIDKey: webpush.VapidKeys;

  getVAPIDKey() {
    if(Notifier.VAPIDKey !== null) return Notifier.VAPIDKey;
    if(fs.existsSync(Notifier.WEBPUSH_VAPID_PRIVATE_KEY_PATH)) return null;
    const privateKey = sshpk.parsePrivateKey(fs.readFileSync(Notifier.WEBPUSH_VAPID_PRIVATE_KEY_PATH), "pem");
    const publicKey = privateKey.toPublic();
    const privateKeyString = (privateKey as any).part.d.data.toString('base64')
    const publicKeyString = (publicKey as any).part.Q.data.toString('base64')

    Notifier.VAPIDKey = webpush.generateVAPIDKeys();
    webpush.setVapidDetails(Notifier.WEBPUSH_SUBJECT, publicKeyString, privateKeyString);
    return Notifier.VAPIDKey;
  }

  async notifyClarificationAnswered(clar: NonNullable<any>, db: PoolConnection, updated = false) {
    const contestants = await db.query(
      clar.disclosed
        ? 'SELECT `id`, `team_id` FROM `contestants`'
        : 'SELECT `id`, `team_id` FROM `contestants` WHERE `team_id` = ?',
      [clar.team_id]
    );

      for (const contestant of contestants) {
        const clarificationMessage = new Notification.ClarificationMessage();
        clarificationMessage.setClarificationId(clar.id);
        clarificationMessage.setOwned(clar.team_id === contestant.team_id);
        clarificationMessage.setUpdated(updated);
        const notification = new Notification();
        notification.setContentClarification(clarificationMessage);
        this.notify(notification, contestant.id, db);
        if (Notifier.VAPIDKey) this.notifyWebpush(notification, contestant.id, db);
      }
  }

  async notifyBenchmarkJobFinished(job, db: PoolConnection) {
    const contestants = await db.query(
      'SELECT `id`, `team_id` FROM `contestants` WHERE `team_id` = ?',
      [job.team_id]
    );

    for (const contestant of contestants) {
      const benchmarkJobMessage = new Notification.BenchmarkJobMessage();
      benchmarkJobMessage.setBenchmarkJobId(job.id);
      const notification = new Notification();
      notification.setContentBenchmarkJob(benchmarkJobMessage);
      await this.notify(notification, contestant.id, db);
      if (Notifier.VAPIDKey) await this.notifyWebpush(notification, contestant.id, db);
    }
  }

  async notify(notification: Notification, contestantId, db) {
    const encodedMessage = Buffer.from(notification.serializeBinary()).toString('base64');
    await db.query(
      'INSERT INTO `notifications` (`contestant_id`, `encoded_message`, `read`, `created_at`, `updated_at`) VALUES (?, ?, FALSE, NOW(6), NOW(6))',
      [contestantId, encodedMessage]
    );
  }

  async notifyWebpush(notification, contestantId, db) {
    const message = Buffer.from(notification.serializeBinary()).toString('base64');
    const subs = await db.query(
      'SELECT * FROM `push_subscriptions` WHERE `contestant_id` = ?',
      [contestantId]
    );

    const requestOpts: webpush.RequestOptions = {
      vapidDetails: {
        subject: Notifier.WEBPUSH_SUBJECT,
        ...Notifier.VAPIDKey
      },
    };

    for (const sub of subs) {
      const pushSubscription: webpush.PushSubscription = {
        endpoint: sub.endpoint,
        keys: {
            p256dh: sub.p256dh,
            auth: sub.auth
        }
      };
      webpush.sendNotification(pushSubscription, message, requestOpts);
    }
  }
}
