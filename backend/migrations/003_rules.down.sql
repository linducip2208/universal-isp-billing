-- 003_rules.down.sql
DELETE FROM automation_rules WHERE name IN (
 'Suspend on overdue + grace expired','Activate on payment confirmed',
 'Alert on device offline','Notify NOC when AP offline > 5 min','Alert on high CPU');
