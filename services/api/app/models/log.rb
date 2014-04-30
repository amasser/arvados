class Log < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :properties, Hash
  before_validation :set_default_event_at
  attr_accessor :object, :object_kind

  api_accessible :user, extend: :common do |t|
    t.add :object_uuid
    t.add :object_owner_uuid
    t.add :object_kind
    t.add :event_at
    t.add :event_type
    t.add :summary
    t.add :properties
  end

  def object_kind
    if k = ArvadosModel::resource_class_for_uuid(object_uuid)
      k.kind
    end
  end

  def fill_object(thing)
    self.object_uuid ||= thing.uuid
    self.object_owner_uuid = thing.owner_uuid
    self.summary ||= "#{self.event_type} of #{thing.uuid}"
    self
  end

  def fill_properties(age, etag_prop, attrs_prop)
    self.properties.merge!({"#{age}_etag" => etag_prop,
                             "#{age}_attributes" => attrs_prop})
  end

  def update_to(thing)
    fill_properties('new', thing.andand.etag, thing.andand.logged_attributes)
    case event_type
    when "create"
      self.event_at = thing.created_at
    when "update"
      self.event_at = thing.modified_at
    when "destroy"
      self.event_at = Time.now
    end
    self
  end

  protected

  def permission_to_create
    true
  end

  def permission_to_update
    current_user.andand.is_admin
  end

  alias_method :permission_to_delete, :permission_to_update

  def set_default_event_at
    self.event_at ||= Time.now
  end

  def log_change(event_type)
    # Don't log changes to logs.
  end

  def ensure_valid_uuids
    # logs can have references to deleted objects
  end

end
