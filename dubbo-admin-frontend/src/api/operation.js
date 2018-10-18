const Operations = [
  {id: 0,
    icon: function (item) {
      return 'visibility'
    },
    tooltip: function (item) {
      return 'View'
    }},
  {id: 1,
    icon: function (item) {
      return 'edit'
    },
    tooltip: function (item) {
      return 'Edit'
    }},
  {id: 2,
    icon: function (item) {
      if (item.enabled) {
        return 'block'
      }
      return 'check_circle_outline'
    },
    tooltip: function (item) {
      if (item.enabled === true) {
        return 'Disable'
      }
      return 'Enable'
    }},
  {id: 3,
    icon: function (item) {
      return 'delete'
    },
    tooltip: function (item) {
      return 'Delete'
    }}
]

export default Operations
