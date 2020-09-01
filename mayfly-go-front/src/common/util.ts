/**
 * 时间字符串转成具体日期，数据库里的时间戳可直接传入转换
 * @author JOU
 * @time   2019-03-31T21:58:06+0800
 * @param  {number}                 timeStr 时间字符串
 * @return {string}                    转换后的具体时间日期
 */
export function time2Date(timeStr: string) {
  if (timeStr === '2100-01-01 00:00:00') {
    return '长期';
  }
  
  const
    ts = new Date(timeStr).getTime() / 1000,
    dateObj = new Date(),
    tsn = Date.parse(dateObj.toString()) / 1000,
    timeGap = tsn - ts,
    oneDayTs = 24 * 60 * 60,
    oneHourTs = 60 * 60,
    oneMinuteTs = 60,
    fillZero = (num: number) => num >= 0 && num < 10 ? ('0' + num) : num.toString(),
    getTimestamp = (dateObj: Date) => Date.parse(dateObj.toString()) / 1000;

  // 未来的时间1天后的，显示“xx天后”
  if (timeGap < -oneDayTs) {
    return Math.floor(-timeGap / oneDayTs) + '天后';
  }

  // 未来不到一天的时间，显示“xx小时后”
  if (timeGap > -oneDayTs && timeGap < -oneHourTs) {
    return Math.floor(-timeGap / oneHourTs) + '小时后';
  }

  // 未来不到一小时的时间，显示“xx分钟后”
  if (timeGap > -oneHourTs && timeGap < 0) {
    return Math.floor(-timeGap / oneMinuteTs) + '小时后';
  }

  // 十分钟前返回“刚刚”
  if (timeGap < (oneMinuteTs * 10)) {
    return '刚刚';
  }

  // 一小时前显示“xx分钟前”
  if (timeGap < oneHourTs) {
    return `${Math.floor(timeGap / oneMinuteTs)}分钟前`;
  }

  // 当天的显示”xx小时前“
  dateObj.setHours(0, 0, 0, 0);
  if (timeGap < tsn - getTimestamp(dateObj)) {
    return `${Math.floor(timeGap / oneHourTs)}小时前`;
  }

  // 昨天显示”昨天 xx:xx“
  const
    date = dateObj.getDate(),
    d = new Date(ts * 1000);
  dateObj.setDate(date - 1);
  if (timeGap < tsn - getTimestamp(dateObj)) {
    return `昨天 ${fillZero(d.getHours())}:${fillZero(d.getMinutes())}`;
  }

  // 前天显示”前天 xx:xx“
  dateObj.setDate(date - 2);
  if (timeGap < tsn - getTimestamp(dateObj)) {
    return `前天 ${fillZero(d.getHours())}:${fillZero(d.getMinutes())}`;
  }

  // 这周显示”这周x xx:xx“
  // 因为上面减了两天，需设置回去
  dateObj.setDate(date);
  let currentDay = dateObj.getDay(), day = d.getDay();
  const weeks = [ '一', '二', '三', '四', '五', '六', '天' ];

  currentDay = currentDay === 0 ? 7 : currentDay;
  day = day === 0 ? 7 : day;
  dateObj.setDate(date - currentDay + 1);
  if (timeGap < tsn - getTimestamp(dateObj)) {
    return `这周${weeks[day - 1]} ${fillZero(d.getHours())}:${fillZero(d.getMinutes())}`;
  }

  // 上周显示”上周x xx:xx“
  dateObj.setDate(date - 6 - currentDay);
  if (timeGap < tsn - getTimestamp(dateObj)) {
    return `上周${weeks[day - 1]} ${fillZero(d.getHours())}:${fillZero(d.getMinutes())}`;
  }

  // 今年再往前的日期则显示”xx-xx xx:xx“（表示xx月xx日 xx点xx分）
  dateObj.setMonth(0, 1);
  if (timeGap < tsn - getTimestamp(dateObj)) {
    return `${fillZero(d.getMonth() + 1)}-${fillZero(d.getDate())} ${fillZero(d.getHours())}:${fillZero(d.getMinutes())}`;
  }

  return `${d.getFullYear()}-${fillZero(d.getMonth() + 1)}-${fillZero(d.getDate())} ${fillZero(d.getHours())}:${fillZero(d.getMinutes())}`;
}