const Menu = [
  { title: 'Service Search', path: '/service', icon: 'search' },
  {
    title: 'Service Governance',
    icon: 'edit',
    group: 'governance',
    items: [
      { title: 'Routing Rule', path: '/governance/routingRule' },
      { title: 'Dynamic Config', path: '/governance/config' },
      { title: 'Access Control', path: '/governance/access' },
      { title: 'Weight Adjust', path: '/governance/weight' },
      { title: 'Load Balance', path: '/governance/loadbalance' }
    ]
  },
  { title: 'Service Test', path: '/test', icon: 'computer', badge: 'feature' },
  { title: 'Service Mock', path: '/mock', icon: 'computer', badge: 'feature' }
]

export default Menu
