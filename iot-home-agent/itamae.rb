file '/boot/config.txt' do
  owner 'root'
  group 'root'
  mode '755'

  action :edit
  block do |content|
    content.gsub!(/^#\W*(dtparam=i2c_arm=on)$/, '\1')
    content << "\ndtparam=i2c_arm=on\n" unless content.match(/^dtparam=i2c_arm=on$/)
  end
end

file '/etc/modules' do
  owner 'root'
  group 'root'
  mode '644'

  action :edit
  block do |content|
    content.gsub!(/^#\W*(i2c-dev)$/, '\1')
    content << "\ni2c-dev\n" unless content.match(/^i2c-dev$/)
  end
end
