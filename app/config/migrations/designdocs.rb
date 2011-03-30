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
