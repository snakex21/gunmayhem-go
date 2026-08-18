defstate = _currentframe;
gotoAndStop(defstate);
this.onRollOver = function()
{
   if(defstate == 2)
   {
      gotoAndStop(3);
   }
};
this.onRollOut = function()
{
   gotoAndStop(defstate);
};
this.onRelease = function()
{
   if(_name == "teamA")
   {
      _parent.team = 1;
   }
   if(_name == "teamB")
   {
      _parent.team = 2;
   }
   _parent.teamA.defstate = 2;
   _parent.teamB.defstate = 2;
   _parent.teamA.gotoAndStop(2);
   _parent.teamB.gotoAndStop(2);
   this.defstate = 4;
   gotoAndStop(4);
};
