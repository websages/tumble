designs = [{
  :name => "_design/items",
  :document => {
    "_id": "_design/items",
    "views": {
        "page": {
            "map": "function(doc) { emit(doc.created_at,doc) }"
        }
    }
  }
]

designs.each { |designdoc|
  # post designdoc to the database
}


#{
#   "page": {
#       "map": "function(doc) { emit(doc.created_at,doc) }"
#   },
#   "image": {
#       "map": "function(doc) { if (doc.type == 'image' ){ emit(doc._id,doc) } }"
#   }
#}
