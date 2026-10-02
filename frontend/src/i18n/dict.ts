export type Lang = 'en' | 'id' | 'ar'
export const dirOf = (l: Lang): 'ltr' | 'rtl' => (l === 'ar' ? 'rtl' : 'ltr')
export const en: Record<string, string> = {
  'nav.dashboard': 'Dashboard', 'nav.noc': 'NOC', 'nav.customers': 'Customers',
  'nav.billing': 'Billing', 'nav.network': 'Network', 'nav.services': 'Services',
  'nav.monitoring': 'Monitoring', 'noc.title': 'Network Operations Center',
  'billing.invoices': 'Invoices', 'billing.payments': 'Payments',
  'common.search': 'Search', 'common.save': 'Save', 'common.status': 'Status',
  'common.actions': 'Actions', 'common.cancel': 'Cancel', 'common.test': 'Test connection',
  'auth.login': 'Sign in', 'auth.username': 'Username', 'auth.password': 'Password',
  'auth.logout': 'Sign out', 'auth.required': 'Please sign in (demo: admin / secret)',
  'lab.title': 'Connection Lab', 'lab.vendor': 'Vendor', 'lab.family': 'Product family',
  'lab.conn': 'Connection type', 'lab.host': 'Host / base URL', 'lab.user': 'Username / API key',
  'lab.pass': 'Password / secret', 'lab.result': 'Result', 'lab.hint': 'Credentials are used for this test only and never stored.',
  'matrix.title': 'Vendor Capability Matrix', 'matrix.status': 'Status',
  'noc.live': 'Live (5s poll)', 'noc.offline': 'API offline — showing seeded demo state',
  'conn.title': 'Connectors',
}
export const id: Record<string, string> = {
  'nav.dashboard': 'Dasbor', 'nav.noc': 'NOC', 'nav.customers': 'Pelanggan',
  'nav.billing': 'Penagihan', 'nav.network': 'Jaringan', 'nav.services': 'Layanan',
  'nav.monitoring': 'Pemantauan', 'noc.title': 'Pusat Operasi Jaringan',
  'billing.invoices': 'Faktur', 'billing.payments': 'Pembayaran',
  'common.search': 'Cari', 'common.save': 'Simpan', 'common.status': 'Status',
  'common.actions': 'Aksi', 'common.cancel': 'Batal', 'common.test': 'Uji koneksi',
  'auth.login': 'Masuk', 'auth.username': 'Nama pengguna', 'auth.password': 'Kata sandi',
  'auth.logout': 'Keluar', 'auth.required': 'Silakan masuk (demo: admin / secret)',
  'lab.title': 'Lab Koneksi', 'lab.vendor': 'Vendor', 'lab.family': 'Keluarga produk',
  'lab.conn': 'Tipe koneksi', 'lab.host': 'Host / URL dasar', 'lab.user': 'Username / API key',
  'lab.pass': 'Password / secret', 'lab.result': 'Hasil', 'lab.hint': 'Kredensial hanya dipakai untuk pengujian ini dan tidak disimpan.',
  'matrix.title': 'Matriks Kapabilitas Vendor', 'matrix.status': 'Status',
  'noc.live': 'Live (poll 5 dtk)', 'noc.offline': 'API offline — menampilkan data demo',
  'conn.title': 'Konektor',
}
// Arabic core coverage (RTL). Covers navigation + common actions; remaining
// keys fall back to English by design (see translate()).
export const ar: Record<string, string> = {
  'nav.dashboard': 'لوحة القيادة', 'nav.noc': 'مركز العمليات', 'nav.customers': 'العملاء',
  'nav.billing': 'الفوترة', 'nav.network': 'الشبكة', 'nav.services': 'الخدمات',
  'nav.monitoring': 'المراقبة', 'noc.title': 'مركز عمليات الشبكة',
  'billing.invoices': 'الفواتير', 'billing.payments': 'المدفوعات',
  'common.search': 'بحث', 'common.save': 'حفظ', 'common.status': 'الحالة',
  'common.actions': 'إجراءات', 'common.cancel': 'إلغاء', 'common.test': 'اختبار الاتصال',
  'auth.login': 'تسجيل الدخول', 'auth.username': 'اسم المستخدم', 'auth.password': 'كلمة المرور',
  'auth.logout': 'تسجيل الخروج',
  'lab.title': 'مختبر الاتصال', 'matrix.title': 'مصفوفة قدرات الموردين',
  'conn.title': 'الموصلات',
}
