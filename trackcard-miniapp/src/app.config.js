export default defineAppConfig({
  pages: [
    'pages/login/index',
    'pages/shipment/create/index',
    'pages/shipment/detail/index',
    'pages/index/index'
  ],
  window: {
    backgroundTextStyle: 'light',
    navigationBarBackgroundColor: '#fff',
    navigationBarTitleText: '全球货运追踪平台',
    navigationBarTextStyle: 'black'
  },
  permission: {
    'scope.userLocation': {
      desc: '你的位置将用于在地图上展示货物与您的距离'
    }
  },
  requiredPrivateInfos: ['getLocation', 'chooseLocation']
})
